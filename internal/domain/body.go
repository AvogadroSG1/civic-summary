// Package domain defines the core types for the civic-summary pipeline.
package domain

// Body represents a government entity whose meetings are processed.
// Bodies are loaded from configuration and are immutable at runtime.
type Body struct {
	Slug            string   `yaml:"slug" mapstructure:"slug"`
	Name            string   `yaml:"name" mapstructure:"name"`
	PlaylistID      string   `yaml:"playlist_id" mapstructure:"playlist_id"`
	VideoSourceURL  string   `yaml:"video_source_url" mapstructure:"video_source_url"`
	OutputSubdir    string   `yaml:"output_subdir" mapstructure:"output_subdir"`
	FilenamePattern string   `yaml:"filename_pattern" mapstructure:"filename_pattern"`
	TitleDateRegex  string   `yaml:"title_date_regex" mapstructure:"title_date_regex"`
	Tags            []string `yaml:"tags" mapstructure:"tags"`
	PromptTemplate  string   `yaml:"prompt_template" mapstructure:"prompt_template"`
	MeetingTypes    []string `yaml:"meeting_types" mapstructure:"meeting_types"`
	Author          string   `yaml:"author" mapstructure:"author"`
	FooterText      string   `yaml:"footer_text" mapstructure:"footer_text"`
}

// DiscoveryURL returns the URL used to discover videos for this body.
// If VideoSourceURL is set, it takes precedence over PlaylistID.
func (b Body) DiscoveryURL() string {
	if b.VideoSourceURL != "" {
		return b.VideoSourceURL
	}
	return "https://www.youtube.com/playlist?list=" + b.PlaylistID
}

// VideoURL returns the full YouTube watch URL for a given video ID.
func (b Body) VideoURL(videoID string) string {
	return "https://www.youtube.com/watch?v=" + videoID
}
