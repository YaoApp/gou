package office

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PptxPresentation represents the main presentation structure
type PptxPresentation struct {
	XMLName    xml.Name       `xml:"presentation"`
	SlideIDLst PptxSlideIDLst `xml:"sldIdLst"`
}

// PptxSlideIDLst represents the list of slide IDs
type PptxSlideIDLst struct {
	XMLName  xml.Name      `xml:"sldIdLst"`
	SlideIds []PptxSlideID `xml:"sldId"`
}

// PptxSlideID represents a slide ID reference
type PptxSlideID struct {
	XMLName xml.Name `xml:"sldId"`
	ID      string   `xml:"id,attr"`
}

// PptxSlide represents a slide
type PptxSlide struct {
	XMLName xml.Name `xml:"sld"`
	CSld    PptxCSld `xml:"cSld"`
}

// PptxCSld represents the common slide data
type PptxCSld struct {
	XMLName xml.Name   `xml:"cSld"`
	SpTree  PptxSpTree `xml:"spTree"`
}

// PptxSpTree represents the shape tree
type PptxSpTree struct {
	XMLName xml.Name    `xml:"spTree"`
	Shapes  []PptxShape `xml:"sp"`
}

// PptxShape represents a shape in the slide
type PptxShape struct {
	XMLName xml.Name   `xml:"sp"`
	TxBody  PptxTxBody `xml:"txBody"`
	NvSpPr  PptxNvSpPr `xml:"nvSpPr"`
}

// PptxNvSpPr represents non-visual shape properties
type PptxNvSpPr struct {
	XMLName xml.Name  `xml:"nvSpPr"`
	CNvPr   PptxCNvPr `xml:"cNvPr"`
}

// PptxCNvPr represents common non-visual properties
type PptxCNvPr struct {
	XMLName xml.Name `xml:"cNvPr"`
	Name    string   `xml:"name,attr"`
}

// PptxTxBody represents text body
type PptxTxBody struct {
	XMLName    xml.Name        `xml:"txBody"`
	Paragraphs []PptxParagraph `xml:"p"`
}

// PptxParagraph represents a paragraph in text body
type PptxParagraph struct {
	XMLName xml.Name  `xml:"p"`
	PPr     PptxPPr   `xml:"pPr"`
	Runs    []PptxRun `xml:"r"`
}

// PptxPPr represents paragraph properties
type PptxPPr struct {
	XMLName xml.Name `xml:"pPr"`
	Level   string   `xml:"lvl,attr"`
}

// PptxRun represents a run of text
type PptxRun struct {
	XMLName xml.Name `xml:"r"`
	RPr     PptxRPr  `xml:"rPr"`
	Text    string   `xml:"t"`
}

// PptxRPr represents run properties
type PptxRPr struct {
	XMLName xml.Name `xml:"rPr"`
	Bold    string   `xml:"b,attr"`
	Italic  string   `xml:"i,attr"`
}

// PptxCoreProperties represents presentation core properties
type PptxCoreProperties struct {
	XMLName     xml.Name `xml:"coreProperties"`
	Title       string   `xml:"title"`
	Subject     string   `xml:"subject"`
	Creator     string   `xml:"creator"`
	Keywords    string   `xml:"keywords"`
	Description string   `xml:"description"`
}

// PptxRelationships represents presentation relationships
type PptxRelationships struct {
	XMLName       xml.Name           `xml:"Relationships"`
	Relationships []PptxRelationship `xml:"Relationship"`
}

// PptxRelationship represents a single relationship
type PptxRelationship struct {
	XMLName xml.Name `xml:"Relationship"`
	ID      string   `xml:"Id,attr"`
	Type    string   `xml:"Type,attr"`
	Target  string   `xml:"Target,attr"`
}

// parsePptx parses a PPTX document and returns the result
func (p *Parser) parsePptx() (*ParseResult, error) {
	result := &ParseResult{
		Metadata: &Metadata{
			MediaRefs: make(map[string]string),
		},
		Media: []Media{},
	}

	// Parse core properties (metadata)
	if err := p.parsePptxCoreProperties(result); err != nil {
		// Non-fatal error, continue parsing
		fmt.Printf("Warning: Could not parse core properties: %v\n", err)
	}

	// Parse presentation content
	if err := p.parsePptxContent(result); err != nil {
		return nil, fmt.Errorf("failed to parse presentation content: %v", err)
	}

	// Extract media files
	media, err := p.extractMedia("ppt/media/")
	if err != nil {
		fmt.Printf("Warning: Could not extract media: %v\n", err)
		result.Media = []Media{} // Ensure it's an empty array, not nil
	} else {
		result.Media = media
	}

	// Parse relationships to map media references
	if err := p.parsePptxRelationships(result); err != nil {
		fmt.Printf("Warning: Could not parse relationships: %v\n", err)
	}

	return result, nil
}

// parsePptxCoreProperties parses presentation metadata
func (p *Parser) parsePptxCoreProperties(result *ParseResult) error {
	data, err := p.readFile("docProps/core.xml")
	if err != nil {
		return err
	}

	var coreProps PptxCoreProperties
	if err := xml.Unmarshal(data, &coreProps); err != nil {
		return err
	}

	result.Metadata.Title = coreProps.Title
	result.Metadata.Subject = coreProps.Subject
	result.Metadata.Author = coreProps.Creator
	result.Metadata.Keywords = coreProps.Keywords

	return nil
}

// parsePptxContent parses the main presentation content
func (p *Parser) parsePptxContent(result *ParseResult) error {
	// First, read the presentation.xml to get slide references
	data, err := p.readFile("ppt/presentation.xml")
	if err != nil {
		return err
	}

	var presentation PptxPresentation
	if err := xml.Unmarshal(data, &presentation); err != nil {
		return err
	}

	// Get slide relationships
	slideRels, err := p.getPptxSlideRelationships()
	if err != nil {
		return err
	}

	var markdown strings.Builder
	var textRanges []TextRange
	currentPos := 0
	slideNum := 1

	// Process each slide
	for _, slideID := range presentation.SlideIDLst.SlideIds {
		slidePath, exists := slideRels[slideID.ID]
		if !exists {
			continue
		}

		// Add slide separator
		markdown.WriteString(fmt.Sprintf("---\n\n## Slide %d\n\n", slideNum))
		slideStartPos := currentPos
		currentPos += len(fmt.Sprintf("---\n\n## Slide %d\n\n", slideNum))

		// Parse slide content
		slideText, slideMarkdown := p.parseSlideContent(slidePath)

		if slideText != "" {
			// Record text range for this slide
			textRanges = append(textRanges, TextRange{
				StartPos: slideStartPos,
				EndPos:   currentPos + len(slideText),
				Page:     slideNum,
				Type:     "slide",
			})

			markdown.WriteString(slideMarkdown)
			markdown.WriteString("\n\n")
			currentPos += len(slideText) + 2
		}

		slideNum++
	}

	result.Markdown = markdown.String()
	result.Metadata.TextRanges = textRanges
	result.Metadata.Pages = slideNum - 1

	return nil
}

// getPptxSlideRelationships gets the relationship mapping for slides
func (p *Parser) getPptxSlideRelationships() (map[string]string, error) {
	data, err := p.readFile("ppt/_rels/presentation.xml.rels")
	if err != nil {
		return nil, err
	}

	var rels PptxRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return nil, err
	}

	slideRels := make(map[string]string)
	for _, rel := range rels.Relationships {
		if strings.Contains(rel.Type, "slide") && !strings.Contains(rel.Type, "slideLayout") {
			slideRels[rel.ID] = "ppt/" + rel.Target
		}
	}
	return slideRels, nil
}

// parseSlideContent parses a single slide's content
func (p *Parser) parseSlideContent(slidePath string) (string, string) {
	data, err := p.readFile(slidePath)
	if err != nil {
		return "", ""
	}

	var slide PptxSlide
	if err := xml.Unmarshal(data, &slide); err != nil {
		return "", ""
	}

	var plainText strings.Builder
	var markdown strings.Builder

	// Process each shape in the slide
	for _, shape := range slide.CSld.SpTree.Shapes {
		shapeText, shapeMarkdown := p.processPptxShape(shape)
		if shapeText != "" {
			plainText.WriteString(shapeText)
			plainText.WriteString("\n")

			markdown.WriteString(shapeMarkdown)
			markdown.WriteString("\n\n")
		}
	}

	return plainText.String(), markdown.String()
}

// processPptxShape processes a shape and extracts text content
func (p *Parser) processPptxShape(shape PptxShape) (string, string) {
	var plainText strings.Builder
	var markdown strings.Builder

	// Determine if this is a title shape
	isTitle := strings.Contains(strings.ToLower(shape.NvSpPr.CNvPr.Name), "title")

	for _, para := range shape.TxBody.Paragraphs {
		paraText, paraMarkdown := p.processPptxParagraph(para)
		if paraText != "" {
			plainText.WriteString(paraText)

			// Apply title formatting if this is a title shape
			if isTitle && paraText != "" {
				// Determine heading level based on paragraph level
				level := 1
				if para.PPr.Level != "" {
					if lvl, err := strconv.Atoi(para.PPr.Level); err == nil {
						level = lvl + 1
						if level > 6 {
							level = 6
						}
					}
				}
				markdown.WriteString(strings.Repeat("#", level) + " " + paraMarkdown)
			} else {
				markdown.WriteString(paraMarkdown)
			}

			markdown.WriteString("\n")
		}
	}

	return plainText.String(), markdown.String()
}

// processPptxParagraph processes a paragraph in a shape
func (p *Parser) processPptxParagraph(para PptxParagraph) (string, string) {
	var plainText strings.Builder
	var markdown strings.Builder

	for _, run := range para.Runs {
		runText := strings.TrimSpace(run.Text)
		if runText != "" {
			plainText.WriteString(runText)

			// Apply formatting
			isBold := run.RPr.Bold == "1"
			isItalic := run.RPr.Italic == "1"

			if isBold && isItalic {
				markdown.WriteString("***" + runText + "***")
			} else if isBold {
				markdown.WriteString("**" + runText + "**")
			} else if isItalic {
				markdown.WriteString("*" + runText + "*")
			} else {
				markdown.WriteString(runText)
			}
		}
	}

	return plainText.String(), markdown.String()
}

// parsePptxRelationships parses presentation relationships for media references
func (p *Parser) parsePptxRelationships(result *ParseResult) error {
	data, err := p.readFile("ppt/_rels/presentation.xml.rels")
	if err != nil {
		return err
	}

	var rels PptxRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return err
	}

	for _, rel := range rels.Relationships {
		if strings.Contains(rel.Type, "image") {
			// Map relationship ID to media file
			result.Metadata.MediaRefs[rel.ID] = rel.Target
		}
	}

	// Also check slide relationships for media
	slideRels, err := p.getPptxSlideRelationships()
	if err != nil {
		return err
	}

	for _, slidePath := range slideRels {
		p.parsePptxSlideRelationships(result, slidePath)
	}

	return nil
}

// parsePptxSlideRelationships parses relationships for a specific slide
func (p *Parser) parsePptxSlideRelationships(result *ParseResult, slidePath string) {
	// Convert slide path to relationships path
	relsPath := strings.Replace(slidePath, ".xml", ".xml.rels", 1)
	relsPath = strings.Replace(relsPath, "ppt/slides/", "ppt/slides/_rels/", 1)

	data, err := p.readFile(relsPath)
	if err != nil {
		return // Non-fatal error
	}

	var rels PptxRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return // Non-fatal error
	}

	for _, rel := range rels.Relationships {
		if strings.Contains(rel.Type, "image") {
			// Map relationship ID to media file
			result.Metadata.MediaRefs[rel.ID] = rel.Target
		}
	}
}

// cleanPptxText removes extra whitespace and normalizes text
func (p *Parser) cleanPptxText(text string) string {
	// Remove extra whitespace
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}
