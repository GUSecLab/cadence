# CPU Profiling

Cadence offers CPU/Memory profiling. Every run of cadence will produce a file called "cadence.prof". To view the profiling result, install graphviz in the system and run 

`go tool pprof -http=:8080 cadence.prof`

The result diagram should appear in your web browser. 

