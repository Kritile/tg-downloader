package usecase

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mediaharvester/tg-downloader/bot/internal/domain"
	"github.com/mediaharvester/tg-downloader/shared/config"
	"github.com/mediaharvester/tg-downloader/shared/models"
)

const (
	formatListTimeout = 30 * time.Second
)

type formatService struct {
	proxyAddr string
}

// NewFormatService creates a new format service
func NewFormatService(proxyAddr string) domain.FormatService {
	return &formatService{
		proxyAddr: proxyAddr,
	}
}

// ListFormats retrieves available formats for a video URL
func (s *formatService) ListFormats(ctx context.Context, url string, source models.VideoSource) ([]models.VideoFormat, error) {
	ctx, cancel := context.WithTimeout(ctx, formatListTimeout)
	defer cancel()

	// Build yt-dlp command with --list-formats
	args := []string{
		"--list-formats",
		"--no-playlist",
		"--dump-json",
		url,
	}

	// Only YouTube traffic is routed through Xray. TikTok and Instagram bypass it.
	useProxy := source == models.SourceYoutube
	if useProxy && s.proxyAddr != "" {
		proxyURL, err := config.NormalizeSocks5Proxy(s.proxyAddr)
		if err != nil {
			return nil, fmt.Errorf("invalid XRAY_SOCKS5_PROXY: %w", err)
		}
		args = append(args, "--proxy", proxyURL)
	}

	// TikTok and Instagram format discovery remains direct; only YouTube uses Xray.

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp list-formats failed: %w, output: %s", err, string(output))
	}

	return s.parseFormatList(string(output))
}

// parseFormatList parses yt-dlp output to extract format information
func (s *formatService) parseFormatList(output string) ([]models.VideoFormat, error) {
	var formats []models.VideoFormat

	// Try to parse JSON output first (from --dump-json)
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(output), &jsonData); err == nil {
		if formatsData, ok := jsonData["formats"].([]interface{}); ok {
			for _, f := range formatsData {
				formatMap, ok := f.(map[string]interface{})
				if !ok {
					continue
				}

				format := s.parseFormatJSON(formatMap)
				if format != nil && s.isVideoFormat(format) {
					formats = append(formats, *format)
				}
			}
		}
	}

	// If JSON parsing failed or no formats found, try parsing text output
	if len(formats) == 0 {
		formats = s.parseFormatText(output)
	}

	// Sort formats by quality (best first)
	sort.Slice(formats, func(i, j int) bool {
		return formats[i].TBR > formats[j].TBR
	})

	return formats, nil
}

// parseFormatJSON parses a format from JSON map
func (s *formatService) parseFormatJSON(formatMap map[string]interface{}) *models.VideoFormat {
	formatID, _ := formatMap["format_id"].(string)
	if formatID == "" {
		return nil
	}

	ext, _ := formatMap["ext"].(string)
	resolution, _ := formatMap["resolution"].(string)
	if resolution == "" {
		height, ok := formatMap["height"].(float64)
		if ok && height > 0 {
			resolution = fmt.Sprintf("%.0fp", height)
		}
	}

	fileSize := int64(0)
	if size, ok := formatMap["filesize"].(float64); ok {
		fileSize = int64(size)
	}

	tbr := 0.0
	if bitrate, ok := formatMap["tbr"].(float64); ok {
		tbr = bitrate
	}

	formatNote, _ := formatMap["format_note"].(string)
	displayName := s.buildDisplayName(formatID, resolution, ext, formatNote)

	return &models.VideoFormat{
		ID:          formatID,
		FormatID:    formatID,
		Extension:   ext,
		Resolution:  resolution,
		FileSize:    fileSize,
		TBR:         tbr,
		FormatNote:  formatNote,
		DisplayName: displayName,
	}
}

// parseFormatText parses yt-dlp text output for format list
func (s *formatService) parseFormatText(output string) []models.VideoFormat {
	var formats []models.VideoFormat

	// Regex to match format lines like:
	// format code  extension  resolution  notes
	formatRegex := regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+(.*)$`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	inFormatsSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Detect start of formats section
		if strings.Contains(line, "format code") || strings.Contains(line, "ID") {
			inFormatsSection = true
			continue
		}

		if !inFormatsSection {
			continue
		}

		// Empty line or new section ends formats
		if line == "" || strings.HasPrefix(line, "[") {
			inFormatsSection = false
			continue
		}

		matches := formatRegex.FindStringSubmatch(line)
		if len(matches) < 5 {
			continue
		}

		formatID := matches[1]
		ext := matches[2]
		resolution := matches[3]
		note := matches[4]

		// Skip audio-only formats
		if resolution == "audio only" {
			continue
		}

		tbr := s.parseBitrate(note)
		displayName := s.buildDisplayName(formatID, resolution, ext, note)

		formats = append(formats, models.VideoFormat{
			ID:          formatID,
			FormatID:    formatID,
			Extension:   ext,
			Resolution:  resolution,
			TBR:         tbr,
			FormatNote:  note,
			DisplayName: displayName,
		})
	}

	return formats
}

// parseBitrate extracts bitrate from format note
func (s *formatService) parseBitrate(note string) float64 {
	// Look for patterns like "250k" or "1.5M"
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)([kK mM])`)
	matches := re.FindStringSubmatch(note)
	if len(matches) < 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(matches[2])
	if unit == "m" {
		value *= 1000
	}

	return value
}

// buildDisplayName creates a human-readable format name
func (s *formatService) buildDisplayName(formatID, resolution, ext, note string) string {
	parts := []string{}

	if resolution != "" && resolution != "unknown" {
		parts = append(parts, resolution)
	}

	if ext != "" && ext != "unknown" {
		parts = append(parts, ext)
	}

	if note != "" {
		// Clean up note
		note = strings.TrimSpace(note)
		if note != "" && note != "-" {
			parts = append(parts, note)
		}
	}

	if len(parts) == 0 {
		return formatID
	}

	return strings.Join(parts, " ")
}

// isVideoFormat checks if format is a video format (not audio-only)
func (s *formatService) isVideoFormat(format *models.VideoFormat) bool {
	if format.Resolution == "" || format.Resolution == "audio only" {
		return false
	}
	return true
}

// SelectFormat validates and returns the selected format
func (s *formatService) SelectFormat(formats []models.VideoFormat, formatID string) (*models.VideoFormat, error) {
	for i := range formats {
		if formats[i].FormatID == formatID {
			return &formats[i], nil
		}
	}
	return nil, domain.ErrDownloadNotFound
}

// GetBestFormat returns the best format for automatic download
func (s *formatService) GetBestFormat(formats []models.VideoFormat) *models.VideoFormat {
	if len(formats) == 0 {
		return nil
	}

	// Formats are already sorted by TBR (bitrate), best first
	// Prefer formats with resolution <= 720p to stay under Telegram limit
	for i := range formats {
		if strings.Contains(formats[i].Resolution, "720") ||
			strings.Contains(formats[i].Resolution, "480") ||
			strings.Contains(formats[i].Resolution, "360") {
			return &formats[i]
		}
	}

	// Return best available if no preferred resolution found
	return &formats[0]
}
