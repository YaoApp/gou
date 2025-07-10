package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// splitPDF implements PDF splitting functionality
func (p *PDF) splitPDF(ctx context.Context, filePath string, config SplitConfig) ([]string, error) {
	// Validate input
	if config.OutputDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	if config.OutputPrefix == "" {
		config.OutputPrefix = "split"
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get PDF info for validation
	info, err := p.GetInfo(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get PDF info: %w", err)
	}

	var outputFiles []string

	// Choose splitting strategy
	if len(config.PageRanges) > 0 {
		// Split by specified page ranges
		outputFiles, err = p.splitByPageRanges(filePath, config, info.PageCount)
	} else if config.PagesPerFile > 0 {
		// Split by pages per file
		outputFiles, err = p.splitByPagesPerFile(filePath, config, info.PageCount)
	} else {
		// Split each page as separate file
		outputFiles, err = p.splitByPages(filePath, config, info.PageCount)
	}

	if err != nil {
		return nil, err
	}

	return outputFiles, nil
}

// splitByPageRanges splits PDF by specified page ranges
func (p *PDF) splitByPageRanges(filePath string, config SplitConfig, _ int) ([]string, error) {
	var outputFiles []string

	for i, pageRange := range config.PageRanges {
		// Create a temporary subdirectory for this range
		tempDir := filepath.Join(config.OutputDir, fmt.Sprintf("range_%d", i+1))
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}

		// Use pdfcpu API to extract pages to temp directory
		err := api.ExtractPagesFile(filePath, tempDir, []string{pageRange}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to split PDF by range %s: %w", pageRange, err)
		}

		// Find generated file and move it to the desired location
		generatedFiles, err := filepath.Glob(filepath.Join(tempDir, "*.pdf"))
		if err != nil || len(generatedFiles) == 0 {
			return nil, fmt.Errorf("no files generated for range %s", pageRange)
		}

		// Move the file to the desired location
		outputPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_%d.pdf", config.OutputPrefix, i+1))
		if err := os.Rename(generatedFiles[0], outputPath); err != nil {
			return nil, fmt.Errorf("failed to move file: %w", err)
		}

		// Clean up temp directory
		os.RemoveAll(tempDir)

		outputFiles = append(outputFiles, outputPath)
	}

	return outputFiles, nil
}

// splitByPagesPerFile splits PDF by number of pages per file
func (p *PDF) splitByPagesPerFile(filePath string, config SplitConfig, totalPages int) ([]string, error) {
	var outputFiles []string
	fileCount := (totalPages + config.PagesPerFile - 1) / config.PagesPerFile

	for i := 0; i < fileCount; i++ {
		startPage := i*config.PagesPerFile + 1
		endPage := (i + 1) * config.PagesPerFile
		if endPage > totalPages {
			endPage = totalPages
		}

		// Create a temporary subdirectory for this batch
		tempDir := filepath.Join(config.OutputDir, fmt.Sprintf("batch_%d", i+1))
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}

		// Extract page range
		pageRange := fmt.Sprintf("%d-%d", startPage, endPage)
		err := api.ExtractPagesFile(filePath, tempDir, []string{pageRange}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to split PDF pages %s: %w", pageRange, err)
		}

		// Find generated file and move it to the desired location
		generatedFiles, err := filepath.Glob(filepath.Join(tempDir, "*.pdf"))
		if err != nil || len(generatedFiles) == 0 {
			return nil, fmt.Errorf("no files generated for pages %s", pageRange)
		}

		// Move the file to the desired location
		outputPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_%d.pdf", config.OutputPrefix, i+1))
		if err := os.Rename(generatedFiles[0], outputPath); err != nil {
			return nil, fmt.Errorf("failed to move file: %w", err)
		}

		// Clean up temp directory
		os.RemoveAll(tempDir)

		outputFiles = append(outputFiles, outputPath)
	}

	return outputFiles, nil
}

// splitByPages splits PDF into individual pages
func (p *PDF) splitByPages(filePath string, config SplitConfig, totalPages int) ([]string, error) {
	// Build page selection array for all pages
	var pages []string
	for i := 1; i <= totalPages; i++ {
		pages = append(pages, fmt.Sprintf("%d", i))
	}

	// Extract all pages - pdfcpu will generate individual files automatically
	err := api.ExtractPagesFile(filePath, config.OutputDir, pages, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to split PDF: %w", err)
	}

	// Build expected output file paths based on pdfcpu naming convention
	// pdfcpu creates files like: {baseName}_page_{pageNr}.pdf
	baseName := filepath.Base(filePath[:len(filePath)-len(filepath.Ext(filePath))])
	var outputFiles []string
	for i := 1; i <= totalPages; i++ {
		outputFileName := fmt.Sprintf("%s_page_%d.pdf", baseName, i)
		outputPath := filepath.Join(config.OutputDir, outputFileName)
		outputFiles = append(outputFiles, outputPath)
	}

	return outputFiles, nil
}
