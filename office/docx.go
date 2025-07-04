package office

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// DocxDocument represents the main document structure
type DocxDocument struct {
	XMLName xml.Name `xml:"document"`
	Body    DocxBody `xml:"body"`
}

// DocxBody represents the document body
type DocxBody struct {
	XMLName    xml.Name        `xml:"body"`
	Paragraphs []DocxParagraph `xml:"p"`
}

// DocxParagraph represents a paragraph in the document
type DocxParagraph struct {
	XMLName xml.Name  `xml:"p"`
	PPr     DocxPPr   `xml:"pPr"`
	Runs    []DocxRun `xml:"r"`
}

// DocxPPr represents paragraph properties
type DocxPPr struct {
	XMLName xml.Name   `xml:"pPr"`
	PStyle  DocxPStyle `xml:"pStyle"`
}

// DocxPStyle represents paragraph style
type DocxPStyle struct {
	XMLName xml.Name `xml:"pStyle"`
	Val     string   `xml:"val,attr"`
}

// DocxRun represents a run of text with formatting
type DocxRun struct {
	XMLName xml.Name    `xml:"r"`
	RPr     DocxRPr     `xml:"rPr"`
	Text    []DocxText  `xml:"t"`
	Drawing DocxDrawing `xml:"drawing"`
}

// DocxRPr represents run properties (formatting)
type DocxRPr struct {
	XMLName xml.Name    `xml:"rPr"`
	Bold    *DocxBold   `xml:"b"`
	Italic  *DocxItalic `xml:"i"`
}

// DocxBold represents bold formatting
type DocxBold struct {
	XMLName xml.Name `xml:"b"`
}

// DocxItalic represents italic formatting
type DocxItalic struct {
	XMLName xml.Name `xml:"i"`
}

// DocxText represents text content
type DocxText struct {
	XMLName xml.Name `xml:"t"`
	Value   string   `xml:",chardata"`
}

// DocxDrawing represents embedded drawings/images
type DocxDrawing struct {
	XMLName xml.Name `xml:"drawing"`
	// We'll expand this for image handling
}

// DocxCoreProperties represents document core properties
type DocxCoreProperties struct {
	XMLName     xml.Name `xml:"coreProperties"`
	Title       string   `xml:"title"`
	Subject     string   `xml:"subject"`
	Creator     string   `xml:"creator"`
	Keywords    string   `xml:"keywords"`
	Description string   `xml:"description"`
}

// DocxRelationships represents document relationships
type DocxRelationships struct {
	XMLName       xml.Name           `xml:"Relationships"`
	Relationships []DocxRelationship `xml:"Relationship"`
}

// DocxRelationship represents a single relationship
type DocxRelationship struct {
	XMLName xml.Name `xml:"Relationship"`
	ID      string   `xml:"Id,attr"`
	Type    string   `xml:"Type,attr"`
	Target  string   `xml:"Target,attr"`
}

// parseDocx parses a DOCX document and returns the result
func (p *Parser) parseDocx() (*ParseResult, error) {
	result := &ParseResult{
		Metadata: &Metadata{
			MediaRefs: make(map[string]string),
		},
		Media: []Media{}, // Always initialize as empty array
	}

	// Parse core properties (metadata)
	if err := p.parseDocxCoreProperties(result); err != nil {
		// Non-fatal error, continue parsing
		fmt.Printf("Warning: Could not parse core properties: %v\n", err)
	}

	// Parse document content
	if err := p.parseDocxContent(result); err != nil {
		return nil, fmt.Errorf("failed to parse document content: %v", err)
	}

	// Extract media files
	media, err := p.extractMedia("word/media/")
	if err != nil {
		fmt.Printf("Warning: Could not extract media: %v\n", err)
		result.Media = []Media{} // Ensure it's an empty array, not nil
	} else {
		result.Media = media
	}

	// Parse relationships to map media references
	if err := p.parseDocxRelationships(result); err != nil {
		fmt.Printf("Warning: Could not parse relationships: %v\n", err)
	}

	return result, nil
}

// parseDocxCoreProperties parses document metadata
func (p *Parser) parseDocxCoreProperties(result *ParseResult) error {
	data, err := p.readFile("docProps/core.xml")
	if err != nil {
		return err
	}

	var coreProps DocxCoreProperties
	if err := xml.Unmarshal(data, &coreProps); err != nil {
		return err
	}

	result.Metadata.Title = coreProps.Title
	result.Metadata.Subject = coreProps.Subject
	result.Metadata.Author = coreProps.Creator
	result.Metadata.Keywords = coreProps.Keywords

	return nil
}

// parseDocxContent parses the main document content
func (p *Parser) parseDocxContent(result *ParseResult) error {
	data, err := p.readFile("word/document.xml")
	if err != nil {
		return err
	}

	var doc DocxDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return err
	}

	var markdown strings.Builder
	var textRanges []TextRange
	currentPos := 0
	pageNum := 1

	for _, para := range doc.Body.Paragraphs {
		paraText, paraMarkdown := p.processParagraph(para)

		if paraText != "" {
			// Record text range
			textRanges = append(textRanges, TextRange{
				StartPos: currentPos,
				EndPos:   currentPos + len(paraText),
				Page:     pageNum,
				Type:     p.getParagraphType(para),
			})

			markdown.WriteString(paraMarkdown)
			markdown.WriteString("\n\n")
			currentPos += len(paraText) + 2 // +2 for newlines
		}
	}

	result.Markdown = markdown.String()
	result.Metadata.TextRanges = textRanges
	result.Metadata.Pages = pageNum

	return nil
}

// processParagraph processes a single paragraph and returns plain text and markdown
func (p *Parser) processParagraph(para DocxParagraph) (string, string) {
	var plainText strings.Builder
	var markdown strings.Builder

	// Check if this is a heading
	isHeading := false
	headingLevel := 0
	if para.PPr.PStyle.Val != "" {
		if strings.HasPrefix(para.PPr.PStyle.Val, "Heading") {
			isHeading = true
			// Extract heading level (e.g., "Heading1" -> 1)
			if len(para.PPr.PStyle.Val) > 7 {
				switch para.PPr.PStyle.Val[7:] {
				case "1":
					headingLevel = 1
				case "2":
					headingLevel = 2
				case "3":
					headingLevel = 3
				case "4":
					headingLevel = 4
				case "5":
					headingLevel = 5
				case "6":
					headingLevel = 6
				default:
					headingLevel = 1
				}
			}
		}
	}

	// Process runs
	for _, run := range para.Runs {
		runText := p.processRun(run)
		plainText.WriteString(runText)

		// Apply formatting
		if run.RPr.Bold != nil && run.RPr.Italic != nil {
			markdown.WriteString("***" + runText + "***")
		} else if run.RPr.Bold != nil {
			markdown.WriteString("**" + runText + "**")
		} else if run.RPr.Italic != nil {
			markdown.WriteString("*" + runText + "*")
		} else {
			markdown.WriteString(runText)
		}
	}

	text := plainText.String()
	md := markdown.String()

	// Apply heading formatting
	if isHeading && text != "" {
		md = strings.Repeat("#", headingLevel) + " " + md
	}

	return text, md
}

// processRun processes a run and extracts text content
func (p *Parser) processRun(run DocxRun) string {
	var text strings.Builder

	for _, t := range run.Text {
		text.WriteString(t.Value)
	}

	return text.String()
}

// getParagraphType determines the type of paragraph
func (p *Parser) getParagraphType(para DocxParagraph) string {
	if para.PPr.PStyle.Val != "" {
		if strings.HasPrefix(para.PPr.PStyle.Val, "Heading") {
			return "heading"
		}
		if strings.Contains(para.PPr.PStyle.Val, "List") {
			return "list"
		}
	}
	return "text"
}

// parseDocxRelationships parses document relationships for media references
func (p *Parser) parseDocxRelationships(result *ParseResult) error {
	data, err := p.readFile("word/_rels/document.xml.rels")
	if err != nil {
		return err
	}

	var rels DocxRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return err
	}

	for _, rel := range rels.Relationships {
		if strings.Contains(rel.Type, "image") {
			// Map relationship ID to media file
			result.Metadata.MediaRefs[rel.ID] = rel.Target
		}
	}

	return nil
}

// cleanText removes extra whitespace and normalizes text
func (p *Parser) cleanText(text string) string {
	// Remove extra whitespace
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}
