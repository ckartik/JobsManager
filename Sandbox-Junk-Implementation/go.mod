module github.com/ckartik/JobManager

require github.com/go-chi/chi/v5 v5.0.3

require (
	github.com/ckaritk/JobsManager/Jobs v0.0.0
	github.com/google/uuid v1.2.0 // indirect
)

replace github.com/ckaritk/JobsManager/Jobs v0.0.0 => ./Jobs
