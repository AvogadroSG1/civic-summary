// Package domain defines the core types for the civic-summary pipeline.
package domain

// Body represents a government entity whose meetings are processed.
// Bodies are loaded from configuration and are immutable at runtime.
type Body struct {
	Slug            string   `yaml:"slug" mapstructure:"slug"`
	Name            string   `yaml:"name" mapstructure:"name"`
	PlaylistID      string   `yaml:"playlist_id" mapstructure:"playlist_id"`
	OutputSubdir    string   `yaml:"output_subdir" mapstructure:"output_subdir"`
	FilenamePattern string   `yaml:"filename_pattern" mapstructure:"filename_pattern"`
	TitleDateRegex  string   `yaml:"title_date_regex" mapstructure:"title_date_regex"`
	Tags            []string `yaml:"tags" mapstructure:"tags"`
	PromptTemplate  string   `yaml:"prompt_template" mapstructure:"prompt_template"`
	MeetingTypes    []string `yaml:"meeting_types" mapstructure:"meeting_types"`
	Author          string   `yaml:"author" mapstructure:"author"`
	FooterText      string   `yaml:"footer_text" mapstructure:"footer_text"`
}

// PlaylistURL returns the full YouTube playlist URL for this body.
func (b Body) PlaylistURL() string {
	return "https://www.youtube.com/playlist?list=" + b.PlaylistID
}

// VideoURL returns the full YouTube watch URL for a given video ID.
func (b Body) VideoURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}
