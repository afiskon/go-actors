# go-actors

Actor model implementation in Golang. See \*\_test.go for usage examples.

```
# Run tests
go test -test.v -race -failfast -count 100 ./...

# Check code coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
