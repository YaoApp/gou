package converter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yaoapp/gou/graphrag/types"
	goupdf "github.com/yaoapp/gou/pdf"

	// Import WebP decoder for image processing
	"compress/gzip"

	// Import WebP decoder for image processing
	_ "golang.org/x/image/webp"
)

// Global queue for OCR processing
var globalQueue chan *ocrTask

// init initializes the global OCR queue and starts the worker
func init() {
	globalQueue = make(chan *ocrTask, 1000) // Buffer for 1000 tasks
	go globalQueueWorker()
}

// OCRMode defines the processing mode for OCR
type OCRMode string

const (
	// OCRModeQueue process pages sequentially
	OCRModeQueue OCRMode = "queue"

	// OCRModeConcurrent process pages concurrently
	OCRModeConcurrent OCRMode = "concurrent"
)

// File type constants
const (
	FileTypeImage = "image"
	FileTypePDF   = "pdf"
)

// OCR is a converter for image and PDF files using OCR
// Processing pipeline:
/*
   1. Check if the input file/stream is an image or PDF file
   2. If not supported, return error
   3. If it's an image file, process directly with vision converter
   4. If it's a PDF file, extract pages as images using PDF library
   5. Process pages with vision converter:
         a. Queue mode: maintain a global queue, process pages sequentially
         b. Concurrent mode: process pages concurrently with concurrency control
   6. Combine results from all pages
   7. Return combined text and metadata including page information
*/
type OCR struct {
	Vision         types.Converter // Vision converter for OCR processing
	Mode           OCRMode         // Processing mode (queue or concurrent)
	MaxConcurrency int             // Maximum concurrent processing (for concurrent mode)
	CompressSize   int64           // Image compression size (max width or height)
	ForceImageMode bool            // Force convert PDF pages to images even if Vision supports PDF

	// PDF processing
	PDFProcessor *goupdf.PDF        // PDF processor instance
	PDFTool      goupdf.ConvertTool // PDF conversion tool type
	PDFDPI       int                // PDF to image conversion DPI
	PDFFormat    string             // PDF to image format (png, jpg)
	PDFQuality   int                // JPEG quality for PDF conversion
}

// OCROption is the configuration for the OCR converter
type OCROption struct {
	Vision         types.Converter `json:"vision,omitempty"`           // Vision converter instance
	Mode           OCRMode         `json:"mode,omitempty"`             // Processing mode
	MaxConcurrency int             `json:"max_concurrency,omitempty"`  // Max concurrent processing
	CompressSize   int64           `json:"compress_size,omitempty"`    // Image compression size (max width or height)
	ForceImageMode bool            `json:"force_image_mode,omitempty"` // Force convert PDF pages to images

	// PDF conversion settings
	PDFTool     goupdf.ConvertTool `json:"pdf_tool,omitempty"`      // PDF conversion tool (pdftoppm, mutool, imagemagick)
	PDFToolPath string             `json:"pdf_tool_path,omitempty"` // Custom path to PDF tool
	PDFDPI      int                `json:"pdf_dpi,omitempty"`       // PDF to image conversion DPI
	PDFFormat   string             `json:"pdf_format,omitempty"`    // PDF to image format (png, jpg)
	PDFQuality  int                `json:"pdf_quality,omitempty"`   // JPEG quality for PDF conversion
}

// PageInfo represents information about a processed page
type PageInfo struct {
	Index       int     `json:"index"`                  // Page index (0-based)
	PageNumber  int     `json:"page_number"`            // Page number (1-based)
	FilePath    string  `json:"file_path"`              // File path of the page image or PDF
	Text        string  `json:"text,omitempty"`         // Extracted text
	Error       string  `json:"error,omitempty"`        // Error message if processing failed
	ProcessTime float64 `json:"process_time,omitempty"` // Processing time in seconds
	IsImageFile bool    `json:"is_image_file"`          // Whether this is an image file (true) or PDF page reference (false)
}

// ocrTask represents a task for OCR processing
type ocrTask struct {
	ctx      context.Context
	pageInfo *PageInfo
	result   chan *PageInfo
	vision   types.Converter // Vision converter for this task
}

// isValidProcessingMode validates if the processing mode is supported
func isValidProcessingMode(mode OCRMode) bool {
	switch mode {
	case OCRModeQueue, OCRModeConcurrent:
		return true
	default:
		return false
	}
}

// NewOCR creates a new OCR converter instance
func NewOCR(option OCROption) (*OCR, error) {
	// Validate required converter
	if option.Vision == nil {
		return nil, errors.New("vision converter is required")
	}

	// Set default processing mode
	processingMode := option.Mode
	if processingMode == "" {
		processingMode = OCRModeConcurrent // Default to concurrent mode
	}

	// Validate processing mode
	if !isValidProcessingMode(processingMode) {
		return nil, errors.New("invalid processing mode: must be 'queue' or 'concurrent'")
	}

	// Set default concurrency
	maxConcurrency := option.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = 4 // Default: 4 concurrent processes
	}

	// Set default compress size
	compressSize := option.CompressSize
	if compressSize == 0 {
		compressSize = 1024 // Default: max 1024px width or height
	}

	// Set PDF processing defaults
	pdfTool := option.PDFTool
	if pdfTool == "" {
		pdfTool = goupdf.ToolPdftoppm // Default to pdftoppm
	}

	pdfDPI := option.PDFDPI
	if pdfDPI == 0 {
		pdfDPI = 150 // Default DPI
	}

	pdfFormat := option.PDFFormat
	if pdfFormat == "" {
		pdfFormat = "jpg" // Default to JPG
	}

	pdfQuality := option.PDFQuality
	if pdfQuality == 0 {
		pdfQuality = 85 // Default JPEG quality
	}

	// Create PDF processor
	pdfProcessor := goupdf.New(goupdf.Options{
		ConvertTool: pdfTool,
		ToolPath:    option.PDFToolPath,
	})

	ocr := &OCR{
		Vision:         option.Vision,
		Mode:           processingMode,
		MaxConcurrency: maxConcurrency,
		CompressSize:   compressSize,
		ForceImageMode: option.ForceImageMode,
		PDFProcessor:   pdfProcessor,
		PDFTool:        pdfTool,
		PDFDPI:         pdfDPI,
		PDFFormat:      pdfFormat,
		PDFQuality:     pdfQuality,
	}

	// Global queue is always available, initialized in init()

	return ocr, nil
}

// Convert converts an image or PDF file to text using OCR
func (o *OCR) Convert(ctx context.Context, file string, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	o.reportProgress(types.ConverterStatusPending, "Starting OCR processing", 0.0, callback...)

	// Check if file is gzipped and decompress if needed
	actualFile, isGzipped, tempFile, err := o.handleGzipFile(file)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to handle gzip file: %v", err), 0.0, callback...)
		return nil, err
	}

	// Clean up temporary file if created
	if tempFile != "" {
		defer os.Remove(tempFile)
	}

	o.reportProgress(types.ConverterStatusPending, "Detecting file type", 0.05, callback...)

	// Detect file type from the actual file (decompressed if needed)
	fileType, err := o.detectFileType(actualFile)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to detect file type: %v", err), 0.0, callback...)
		return nil, err
	}

	var pages []PageInfo
	switch fileType {
	case FileTypeImage:
		o.reportProgress(types.ConverterStatusPending, "Processing image", 0.1, callback...)

		// For single images, apply compression if needed
		compressedFile, err := o.compressImageFile(actualFile)
		if err != nil {
			o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to compress image: %v", err), 0.0, callback...)
			return nil, err
		}

		pages = []PageInfo{{
			Index:       0,
			PageNumber:  1,
			FilePath:    compressedFile,
			IsImageFile: true,
		}}

	case FileTypePDF:
		// PDF processing
		if o.ForceImageMode {
			o.reportProgress(types.ConverterStatusPending, "Converting PDF pages to images", 0.1, callback...)
			pages, err = o.extractPDFPages(ctx, actualFile)
			if err != nil {
				o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to extract PDF pages: %v", err), 0.0, callback...)
				return nil, err
			}
		} else {
			o.reportProgress(types.ConverterStatusPending, "Analyzing PDF pages", 0.1, callback...)
			// Get PDF page count for direct processing
			pages, err = o.preparePDFPages(actualFile)
			if err != nil {
				o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to analyze PDF pages: %v", err), 0.0, callback...)
				return nil, err
			}
		}

	default:
		o.reportProgress(types.ConverterStatusError, "File is not an image or PDF", 0.0, callback...)
		return nil, errors.New("file is not an image or PDF file")
	}

	o.reportProgress(types.ConverterStatusPending, "Processing pages with OCR", 0.3, callback...)

	// Process pages based on processing mode
	var processedPages []PageInfo
	if o.Mode == OCRModeQueue {
		processedPages, err = o.processPagesQueue(ctx, pages, callback...)
	} else {
		processedPages, err = o.processPagesConcurrent(ctx, pages, callback...)
	}

	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to process pages: %v", err), 0.0, callback...)
		return nil, err
	}

	o.reportProgress(types.ConverterStatusPending, "Combining results", 0.9, callback...)

	// Combine results
	result := o.combineResults(processedPages, fileType)

	// Add gzip information to metadata
	if result.Metadata != nil {
		result.Metadata["gzipped"] = isGzipped
	}

	// Cleanup temporary page files
	switch fileType {
	case FileTypePDF:
		if o.ForceImageMode {
			for _, page := range pages {
				if page.FilePath != actualFile {
					os.Remove(page.FilePath)
				}
			}
		}
	case FileTypeImage:
		// Cleanup compressed image file if it's different from original
		for _, page := range pages {
			if page.FilePath != actualFile {
				os.Remove(page.FilePath)
			}
		}
	}

	o.reportProgress(types.ConverterStatusSuccess, "OCR conversion completed", 1.0, callback...)
	return result, nil
}

// ConvertStream converts an image or PDF stream to text using OCR
func (o *OCR) ConvertStream(ctx context.Context, stream io.ReadSeeker, callback ...types.ConverterProgress) (*types.ConvertResult, error) {
	o.reportProgress(types.ConverterStatusPending, "Processing stream", 0.0, callback...)

	// Check if gzipped and decompress if needed
	reader, isGzipped, err := o.handleGzipStream(stream)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to handle gzip stream: %v", err), 0.0, callback...)
		return nil, err
	}

	// Save stream to temporary file for processing
	tempFile, err := o.saveStreamToTempFile(reader)
	if err != nil {
		o.reportProgress(types.ConverterStatusError, fmt.Sprintf("Failed to save stream: %v", err), 0.0, callback...)
		return nil, err
	}
	defer func() {
		os.Remove(tempFile)
	}()

	o.reportProgress(types.ConverterStatusPending, "Stream saved, processing as file", 0.1, callback...)

	// Use Convert to process the temporary file
	result, err := o.Convert(ctx, tempFile, callback...)
	if err != nil {
		return nil, err
	}

	// Add gzip information to metadata
	if result.Metadata != nil {
		result.Metadata["gzipped"] = isGzipped
	}

	return result, nil
}

// handleGzipStream checks if the stream is gzipped and decompresses if needed
func (o *OCR) handleGzipStream(stream io.ReadSeeker) (io.Reader, bool, error) {
	// Check if gzipped
	peekBuffer := make([]byte, 2)
	n, err := stream.Read(peekBuffer)
	if err != nil && err != io.EOF {
		return nil, false, fmt.Errorf("failed to peek stream: %w", err)
	}

	// Reset stream position
	_, err = stream.Seek(0, io.SeekStart)
	if err != nil {
		return nil, false, fmt.Errorf("failed to reset stream: %w", err)
	}

	// Check for gzip magic bytes (0x1f, 0x8b)
	if n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b {
		gzReader, err := gzip.NewReader(stream)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzReader, true, nil
	}

	return stream, false, nil
}

// saveStreamToTempFile saves the stream to a temporary file
func (o *OCR) saveStreamToTempFile(reader io.Reader) (string, error) {
	tempFile, err := os.CreateTemp(os.TempDir(), "ocr_input_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, reader)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to copy stream to temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// handleGzipFile checks if the file is gzipped and decompresses if needed
func (o *OCR) handleGzipFile(filePath string) (string, bool, string, error) {
	// Open the file to check if it's gzipped
	file, err := os.Open(filePath)
	if err != nil {
		return "", false, "", fmt.Errorf("failed to open file for gzip detection: %w", err)
	}
	defer file.Close()

	// Read the first few bytes to check for gzip magic bytes
	peekBuffer := make([]byte, 2)
	n, err := file.Read(peekBuffer)
	if err != nil && err != io.EOF {
		return "", false, "", fmt.Errorf("failed to peek file header: %w", err)
	}

	// Reset file position
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return "", false, "", fmt.Errorf("failed to reset file position: %w", err)
	}

	// Check for gzip magic bytes (0x1f, 0x8b)
	if n >= 2 && peekBuffer[0] == 0x1f && peekBuffer[1] == 0x8b {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return "", false, "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()

		// Create a temporary file to write the decompressed content
		tempFile, err := os.CreateTemp(os.TempDir(), "ocr_input_*")
		if err != nil {
			return "", false, "", fmt.Errorf("failed to create temp file for decompressed content: %w", err)
		}
		defer tempFile.Close()

		_, err = io.Copy(tempFile, gzReader)
		if err != nil {
			os.Remove(tempFile.Name())
			return "", false, "", fmt.Errorf("failed to copy decompressed content to temp file: %w", err)
		}

		return tempFile.Name(), true, tempFile.Name(), nil // Return the temporary file path as both actualFile and tempFile
	}

	return filePath, false, "", nil // Return original file path
}

// detectFileType detects if the file is an image or PDF
func (o *OCR) detectFileType(filePath string) (string, error) {
	// First try to open the file to check if it exists
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for detection: %w", err)
	}
	defer file.Close()

	// Read file header for magic byte detection
	buffer := make([]byte, 8)
	n, err := file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}

	if n < 2 {
		return "", errors.New("file too small to determine type")
	}

	// Check PDF magic bytes first (most reliable)
	if n >= 4 && string(buffer[0:4]) == "%PDF" {
		return FileTypePDF, nil
	}

	// Check common image magic bytes
	// JPEG
	if n >= 2 && buffer[0] == 0xFF && buffer[1] == 0xD8 {
		return FileTypeImage, nil
	}
	// PNG
	if n >= 8 && buffer[0] == 0x89 && string(buffer[1:4]) == "PNG" {
		return FileTypeImage, nil
	}
	// GIF
	if n >= 6 && (string(buffer[0:6]) == "GIF87a" || string(buffer[0:6]) == "GIF89a") {
		return FileTypeImage, nil
	}
	// WebP (RIFF container with WEBP signature)
	if n >= 4 && string(buffer[0:4]) == "RIFF" {
		// For WebP, we need to read more to check for WEBP signature
		// But for now, assume RIFF files are WebP since we only handle images
		return FileTypeImage, nil
	}
	// BMP
	if n >= 2 && buffer[0] == 0x42 && buffer[1] == 0x4D {
		return FileTypeImage, nil
	}
	// TIFF
	if n >= 4 && ((buffer[0] == 0x49 && buffer[1] == 0x49 && buffer[2] == 0x2A && buffer[3] == 0x00) ||
		(buffer[0] == 0x4D && buffer[1] == 0x4D && buffer[2] == 0x00 && buffer[3] == 0x2A)) {
		return FileTypeImage, nil
	}

	// If magic bytes don't match, fall back to file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif", ".webp":
		return FileTypeImage, nil
	case ".pdf":
		return FileTypePDF, nil
	}

	return "", errors.New("file is not a supported image or PDF format")
}

// extractPDFPages extracts pages from PDF file as images
func (o *OCR) extractPDFPages(ctx context.Context, pdfPath string) ([]PageInfo, error) {
	// Create temporary directory for PDF pages
	pagesDir := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_pages_%d", time.Now().UnixNano()))
	err := os.MkdirAll(pagesDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create pages directory: %w", err)
	}

	// Configure PDF to image conversion
	convertConfig := goupdf.ConvertConfig{
		OutputDir:    pagesDir,
		OutputPrefix: "page",
		Format:       o.PDFFormat,
		DPI:          o.PDFDPI,
		Quality:      o.PDFQuality,
		PageRange:    "all", // Convert all pages
	}

	// Convert PDF pages to images
	imageFiles, err := o.PDFProcessor.Convert(ctx, pdfPath, convertConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert PDF pages to images: %w", err)
	}

	var pages []PageInfo
	for i, imageFile := range imageFiles {
		// Apply compression to extracted page if needed
		finalPath := imageFile
		if o.CompressSize > 0 {
			compressedPath, compressErr := o.compressImageFile(imageFile)
			if compressErr != nil {
				// If compression fails, use original file but log warning
				finalPath = imageFile
			} else {
				finalPath = compressedPath
				// Remove original uncompressed file if compression succeeded
				if compressedPath != imageFile {
					os.Remove(imageFile)
				}
			}
		}

		pages = append(pages, PageInfo{
			Index:       i,
			PageNumber:  i + 1,
			FilePath:    finalPath,
			IsImageFile: true, // This is an extracted image file
		})
	}

	return pages, nil
}

// preparePDFPages prepares page information for direct PDF processing
func (o *OCR) preparePDFPages(pdfPath string) ([]PageInfo, error) {
	// Get PDF page count using our PDF library
	info, err := o.PDFProcessor.GetInfo(context.Background(), pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get PDF info: %w", err)
	}

	var pages []PageInfo
	for i := 1; i <= info.PageCount; i++ {
		pages = append(pages, PageInfo{
			Index:       i - 1,
			PageNumber:  i,
			FilePath:    pdfPath, // All pages reference the same PDF file
			IsImageFile: false,   // This is a PDF page reference
		})
	}

	return pages, nil
}

// createSinglePagePDF creates a temporary PDF file containing only the specified page
func (o *OCR) createSinglePagePDF(srcPDF string, pageNumber int) (string, error) {
	// Create temporary directory for single page PDF
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("pdf_single_page_%d", time.Now().UnixNano()))
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Configure split to extract only the specific page
	splitConfig := goupdf.SplitConfig{
		OutputDir:    tempDir,
		OutputPrefix: fmt.Sprintf("page_%d", pageNumber),
		PageRanges:   []string{fmt.Sprintf("%d", pageNumber)}, // Extract only this page
	}

	// Split the PDF to extract the specific page
	outputFiles, err := o.PDFProcessor.Split(context.Background(), srcPDF, splitConfig)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to extract page %d: %w", pageNumber, err)
	}

	if len(outputFiles) == 0 {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("no output file generated for page %d", pageNumber)
	}

	return outputFiles[0], nil
}

// compressImageFile compresses an image file if it exceeds the compression size
func (o *OCR) compressImageFile(inputPath string) (string, error) {
	// If no compression size set, return original file
	if o.CompressSize <= 0 {
		return inputPath, nil
	}

	// Read original file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Check image dimensions
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	maxSize := int(o.CompressSize)

	// Check if compression is needed
	if width <= maxSize && height <= maxSize {
		return inputPath, nil
	}

	// Calculate new dimensions maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = maxSize
		newHeight = int(float64(height) * (float64(maxSize) / float64(width)))
	} else {
		newHeight = maxSize
		newWidth = int(float64(width) * (float64(maxSize) / float64(height)))
	}

	// Create new image with new dimensions using simple scaling
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * float64(width) / float64(newWidth))
			srcY := int(float64(y) * float64(height) / float64(newHeight))
			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Create compressed file
	compressedPath := filepath.Join(os.TempDir(), fmt.Sprintf("compressed_%d_%s", time.Now().UnixNano(), filepath.Base(inputPath)))
	compressedFile, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create compressed file: %w", err)
	}
	defer compressedFile.Close()

	// Encode with appropriate format
	switch strings.ToLower(format) {
	case "png":
		err = png.Encode(compressedFile, newImg)
	case "gif":
		err = gif.Encode(compressedFile, newImg, nil)
	case "jpeg", "jpg":
		err = jpeg.Encode(compressedFile, newImg, &jpeg.Options{Quality: 85})
	default:
		// Default to JPEG for all other formats (including WebP, etc.)
		err = jpeg.Encode(compressedFile, newImg, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		os.Remove(compressedPath)
		return "", fmt.Errorf("failed to encode compressed image: %w", err)
	}

	return compressedPath, nil
}

// globalQueueWorker processes tasks from the global queue sequentially
func globalQueueWorker() {
	for task := range globalQueue {
		if task.ctx.Err() != nil {
			// Context canceled, send error result
			task.result <- &PageInfo{
				Index:      task.pageInfo.Index,
				PageNumber: task.pageInfo.PageNumber,
				FilePath:   task.pageInfo.FilePath,
				Error:      task.ctx.Err().Error(),
			}
			continue
		}

		// Process the page
		startTime := time.Now()
		result := *task.pageInfo

		var filePath string
		var tempFile string

		// Determine the file path to process
		if task.pageInfo.IsImageFile {
			// For image files, process directly
			filePath = task.pageInfo.FilePath
		} else {
			// For PDF page references, create a temporary single-page PDF
			ocr := &OCR{} // Create a temporary OCR instance for the helper method
			var err error
			tempFile, err = ocr.createSinglePagePDF(task.pageInfo.FilePath, task.pageInfo.PageNumber)
			if err != nil {
				result.Error = fmt.Sprintf("failed to create single page PDF: %v", err)
				result.ProcessTime = time.Since(startTime).Seconds()
				task.result <- &result
				continue
			}
			filePath = tempFile
			defer os.Remove(tempFile) // Clean up temporary file
		}

		// Call vision converter
		convertResult, err := task.vision.Convert(task.ctx, filePath)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Text = convertResult.Text
		}
		result.ProcessTime = time.Since(startTime).Seconds()

		select {
		case task.result <- &result:
		case <-task.ctx.Done():
		}
	}
}

// processPagesQueue processes pages using global queue mode
func (o *OCR) processPagesQueue(ctx context.Context, pages []PageInfo, callback ...types.ConverterProgress) ([]PageInfo, error) {
	results := make([]PageInfo, len(pages))
	resultChans := make([]chan *PageInfo, len(pages))

	// Submit tasks to global queue
	for i := range pages {
		resultChans[i] = make(chan *PageInfo, 1)
		task := &ocrTask{
			ctx:      ctx,
			pageInfo: &pages[i],
			result:   resultChans[i],
			vision:   o.Vision,
		}

		select {
		case globalQueue <- task:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Collect results
	for i := range pages {
		select {
		case result := <-resultChans[i]:
			results[i] = *result

			// Report progress
			completed := i + 1
			progress := 0.3 + (float64(completed)/float64(len(pages)))*0.6
			o.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Processed %d/%d pages", completed, len(pages)), progress, callback...)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

// processPagesConcurrent processes pages using concurrent mode
func (o *OCR) processPagesConcurrent(ctx context.Context, pages []PageInfo, callback ...types.ConverterProgress) ([]PageInfo, error) {
	results := make([]PageInfo, len(pages))
	semaphore := make(chan struct{}, o.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, page := range pages {
		wg.Add(1)
		go func(index int, pageInfo PageInfo) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Process page
			startTime := time.Now()
			result := o.processPage(ctx, pageInfo)
			result.ProcessTime = time.Since(startTime).Seconds()

			mu.Lock()
			results[index] = result

			// Report progress
			completed := 0
			for _, r := range results {
				if r.Text != "" || r.Error != "" {
					completed++
				}
			}
			progress := 0.3 + (float64(completed)/float64(len(pages)))*0.6
			o.reportProgress(types.ConverterStatusPending, fmt.Sprintf("Processed %d/%d pages", completed, len(pages)), progress, callback...)
			mu.Unlock()
		}(i, page)
	}

	wg.Wait()
	return results, nil
}

// processPage processes a single page using vision converter
func (o *OCR) processPage(ctx context.Context, pageInfo PageInfo) PageInfo {
	result := pageInfo

	var filePath string
	var tempFile string

	// Determine the file path to process
	if pageInfo.IsImageFile {
		// For image files, process directly
		filePath = pageInfo.FilePath
	} else {
		// For PDF page references, create a temporary single-page PDF
		var err error
		tempFile, err = o.createSinglePagePDF(pageInfo.FilePath, pageInfo.PageNumber)
		if err != nil {
			result.Error = fmt.Sprintf("failed to create single page PDF: %v", err)
			return result
		}
		filePath = tempFile
		defer os.Remove(tempFile) // Clean up temporary file
	}

	// Call vision converter with OCR prompt
	convertResult, err := o.Vision.Convert(ctx, filePath)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Text = convertResult.Text
	return result
}

// combineResults combines processed pages into final result
func (o *OCR) combineResults(pages []PageInfo, fileType string) *types.ConvertResult {
	var combinedText strings.Builder
	successfulPages := 0
	totalProcessTime := 0.0

	for _, page := range pages {
		if page.Text != "" {
			if fileType == FileTypePDF && (o.ForceImageMode || len(pages) > 1) {
				// Only add page numbers if we have multiple pages or forced image mode
				combinedText.WriteString(fmt.Sprintf("Page %d:\n", page.PageNumber))
			}
			combinedText.WriteString(page.Text)
			combinedText.WriteString("\n\n")
			successfulPages++
		}
		totalProcessTime += page.ProcessTime
	}

	// Create metadata
	metadata := map[string]interface{}{
		"source_type":        fileType,
		"total_pages":        len(pages),
		"successful_pages":   successfulPages,
		"processing_mode":    string(o.Mode),
		"max_concurrency":    o.MaxConcurrency,
		"total_process_time": totalProcessTime,
		"text_length":        combinedText.Len(),
		"compress_size":      o.CompressSize,
		"force_image_mode":   o.ForceImageMode,
	}

	// Add processing method information
	switch fileType {
	case FileTypePDF:
		if o.ForceImageMode {
			metadata["pdf_processing_method"] = "pdf_library_image_extraction"
			metadata["pdf_tool"] = string(o.PDFTool)
			metadata["pdf_dpi"] = o.PDFDPI
			metadata["pdf_format"] = o.PDFFormat
			metadata["pdf_quality"] = o.PDFQuality
		} else {
			metadata["pdf_processing_method"] = "direct_processing"
		}
	case FileTypeImage:
		metadata["image_compressed"] = o.CompressSize > 0
	}

	// Include per-page information
	pageInfos := make([]map[string]interface{}, len(pages))
	for i, page := range pages {
		pageInfos[i] = map[string]interface{}{
			"page_number":  page.PageNumber,
			"text_length":  len(page.Text),
			"process_time": page.ProcessTime,
			"success":      page.Text != "",
		}
		if page.Error != "" {
			pageInfos[i]["error"] = page.Error
		}
	}
	metadata["pages"] = pageInfos

	return &types.ConvertResult{
		Text:     strings.TrimSpace(combinedText.String()),
		Metadata: metadata,
	}
}

// reportProgress reports conversion progress
func (o *OCR) reportProgress(status types.ConverterStatus, message string, progress float64, callbacks ...types.ConverterProgress) {
	if len(callbacks) == 0 {
		return
	}

	payload := types.ConverterPayload{
		Status:   status,
		Message:  message,
		Progress: progress,
	}

	for _, callback := range callbacks {
		if callback != nil {
			callback(status, payload)
		}
	}
}

// Close cleans up resources
func (o *OCR) Close() error {
	// No instance-specific resources to clean up
	// Global queue is shared across all instances
	return nil
}
