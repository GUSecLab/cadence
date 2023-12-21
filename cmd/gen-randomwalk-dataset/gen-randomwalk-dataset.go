package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"sync"

	"github.com/akamensky/argparse"
	logger "github.com/sirupsen/logrus"
)

type world_info struct {
	sim_max_time      int
	time_step         int
	max_lat           float64
	max_lon           float64
	max_step_size_lat float64
	max_step_size_lon float64
}

type position struct {
	lat float64
	lon float64
}

func (pos *position) String() string {
	return fmt.Sprintf("(%v,%v)", pos.lat, pos.lon)
}
func (pos *position) rand(world *world_info) {
	pos.lat = rand.Float64() * world.max_lat
	pos.lon = rand.Float64() * world.max_lon
}
func (pos *position) move_rel(lat, lon float64) {
	pos.lat += lat
	pos.lon += lon
}

type event struct {
	user     int
	sim_time int
	pos      *position
}

var log *logger.Logger

func event_recorder(expected_events int, output_file string, ch chan *event, done chan bool) {
	events := make([]*event, 0, expected_events)
	for e := range ch {
		events = append(events, e)
		log.Debugf("at time %v, user %v at position %v", e.sim_time, e.user, e.pos)
	}
	log.Info("sorting events")
	// sort'em!
	sort.Slice(events, func(i, j int) bool {
		return events[i].sim_time < events[j].sim_time
	})
	// and dump'em!
	log.Info("dumping events to file")
	f, err := os.Create(output_file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	for _, e := range events {
		f.WriteString(fmt.Sprintf("%v,%v,%v,%v\n", e.sim_time, e.user, e.pos.lat, e.pos.lon))
	}
	done <- true
}

func take_a_walk(user int, world *world_info, ch chan *event) {
	var pos position
	pos.rand(world)

	log.Debugf("user %v is taking a walk, beginning at %v", user, pos)
	for sim_time := 0; sim_time < world.sim_max_time; sim_time += world.time_step {

		step_lat := rand.Float64() * world.max_step_size_lat
		if rand.Int()%2 == 0 {
			step_lat *= -1.0
		}
		step_lon := rand.Float64() * world.max_step_size_lon
		if rand.Int()%2 == 0 {
			step_lon *= -1.0
		}
		pos.move_rel(step_lat, step_lon)
		//log.Debugf("at time %v, user %v is at %v", sim_time, user, pos)
		ch <- &event{
			user:     user,
			sim_time: sim_time,
			pos:      &pos,
		}
	}
}

func feet_to_degree(feet float64) float64 {
	return feet / 364000.0
}

func main() {

	log = logger.New()
	log.SetLevel(logger.InfoLevel)

	parser := argparse.NewParser("gen-randomwalk-data", "produces a random walk dataset")

	output_file := parser.String("f", "output", &argparse.Options{
		Help:     "file to (over)write",
		Required: true,
	})
	num_users := parser.Int("n", "numusers", &argparse.Options{
		Help:     "number of users to simulate",
		Required: true,
	})
	sim_time_max := parser.Int("t", "simtime", &argparse.Options{
		Help:     "simulation time (in seconds)",
		Required: true,
	})
	time_step := parser.Int("s", "timestep", &argparse.Options{
		Help:    "time between movements (in seconds)",
		Default: 60,
	})
	max_lat := parser.Float("l", "maxlat", &argparse.Options{
		Help:    "maximum latitude",
		Default: 10.0,
	})
	max_lon := parser.Float("L", "maxlon", &argparse.Options{
		Help:    "maximum longitude",
		Default: 10.0,
	})
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		panic("invalid usage")
	}

	world := world_info{
		sim_max_time:      *sim_time_max,
		time_step:         *time_step,
		max_lat:           *max_lat,
		max_lon:           *max_lon,
		max_step_size_lat: feet_to_degree(100.0),
		max_step_size_lon: feet_to_degree(100.0),
	}

	ch := make(chan *event)
	done := make(chan bool)

	go event_recorder(
		*num_users*world.sim_max_time,
		*output_file,
		ch, done)

	var wg sync.WaitGroup
	for user := 0; user < *num_users; user++ {
		wg.Add(1)
		go func(u int) {
			defer wg.Done()
			take_a_walk(u, &world, ch)
		}(user)
	}

	// wait for all of the go-routines to finish
	wg.Wait()
	close(ch)
	log.Info("all walkers done")

	// wait for the event recorder to finish
	<-done
}
