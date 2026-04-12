module github.com/joakimcarlsson/ai/integrations/postgres

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/joakimcarlsson/ai v0.18.2
	github.com/lib/pq v1.12.3
)

replace github.com/joakimcarlsson/ai => ../..
