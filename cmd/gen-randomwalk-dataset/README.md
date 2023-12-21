# A script to generate a VERY random and extremely unrealistic human mobility dataset

[Micah Sherr](https://micahsherr.com)

## Building

```
go mod tidy
go get -d .
go build .
```

## Running

```
usage: gen-randomwalk-data [-h|--help] -f|--output "<value>" -n|--numusers
                           <integer> -t|--simtime <integer> [-s|--timestep
                           <integer>] [-l|--maxlat <float>] [-L|--maxlon
                           <float>]

                           produces a random walk dataset

Arguments:

  -h  --help      Print help information
  -f  --output    file to (over)write
  -n  --numusers  number of users to simulate
  -t  --simtime   simulation time (in seconds)
  -s  --timestep  time between movements (in seconds). Default: 60
  -l  --maxlat    maximum latitude. Default: 10
  -L  --maxlon    maximum longitude. Default: 10
```
