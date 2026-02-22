package domain

// PipelineStage represents a stage in the processing pipeline.
type PipelineStage string

const (
	StageDiscovery     PipelineStage = "discovery"
	StageTranscription PipelineStage = "transcription"
	StageAnalysis      PipelineStage = "analysis"
	StageCrossRef      PipelineStage = "crossref"
	StageValidation    PipelineStage = "validation"
)

// PipelineResult tracks the outcome of processing a single meeting.
type PipelineResult struct {
	Meeting    Meeting
	Stage      PipelineStage
	Success    bool
	Error      error
	OutputPath string
}

// ProcessingStats tracks aggregate pipeline statistics.
type ProcessingStats struct {
	Discovered  int
	Skipped     int
	Processed   int
	Failed      int
	Quarantined int
}
