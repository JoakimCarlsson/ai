module github.com/joakimcarlsson/ai/integrations/postgres

go 1.24.2

require (
	github.com/google/uuid v1.6.0
	github.com/joakimcarlsson/ai v0.0.0
	github.com/lib/pq v1.10.9
)

replace github.com/joakimcarlsson/ai => ../..
