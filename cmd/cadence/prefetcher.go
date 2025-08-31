/*
This file defines a database prefetcher that grabs lots of records from the
database and stores it in memory.  If more records are needed, it fetches those
in large batches and caches the result.
*/

package main

import (
	model "marathon-sim/datamodel"
	"sync"
)

// let's decide arbitrarily how much data we should actually prefetch
const prefetchBatchSize = 5000

var prefetchDBLock sync.Mutex

type prefetcherBin struct {
	eventList []*model.Event
	done      bool
}

type prefetcher struct {
	prefetchedDataMap *sync.Map
}

func (pb *prefetcherBin) init() {
	pb.eventList = make([]*model.Event, 0, prefetchBatchSize)
	pb.done = false
}

// Fetches more events for the given node, starting at the entry *at or before* the
// specified time. Returns false iff there are no more events to fetch
func (pb *prefetcherBin) fetchMoreEvents(nodeId model.NodeId, t float64) (bool, error) {
	// fetch a batch (of size `prefetchBatchSize`) of events from the DB
	var bufEventList []*model.Event

	// it's better for the database if we serialize this
	prefetchDBLock.Lock()
	defer prefetchDBLock.Unlock()

	// first, figure out the time of the entry at or before t
	rows, err := model.DB.Limit(1).Table("events").Select("max(time)").Where("dataset_name=? and node=? and time<=?", dataset.DatasetName, nodeId, t).Rows()
	if err != nil {
		log.Warn("DB 'entry-before' fetch failed: ", err)
		return true, err
	}
	defer rows.Close()
	if !rows.Next() {
		// log.Info("failed in max time")
		pb.done = true
		return false, nil
	}
	var timeStart float64
	if err := rows.Scan(&timeStart); err != nil {
		log.Debug("DB scan failed: ", err)
		pb.done = true
		// log.Infof("failed in time start scan %v for time %v", err, t)
		return false, nil
	}
	//log.Debugf("for node %v, found time %f that is closest to %f without going over", nodeId, timeStart, t)

	// now that we know where to start looking, fetch more events!
	if r := model.DB.Limit(prefetchBatchSize).Find(&bufEventList, "dataset_name = ? and node = ? and time >= ?", dataset.DatasetName, nodeId, timeStart).Order("time"); r.Error != nil {
		log.Warn("DB fetch failed: ", r.Error)
		return true, r.Error
	}
	for _, e := range bufEventList {
		l := len(pb.eventList)
		if l == 0 || pb.eventList[l-1].Time < e.Time {
			pb.eventList = append(pb.eventList, e)
		}
	}

	// if there's no more events for this node, let's record that.  Note that we
	// need 2 events -- the one before this time and the one after it.
	if len(pb.eventList) < 2 {
		pb.done = true
		return false, nil
	}
	return true, nil
}

// remove all events from the eventList before time `t`
func (pb *prefetcherBin) cullEventsBeforeTime(t float64) {
	for i, event := range pb.eventList {
		if event.Time >= t {
			// stop here, and cut all entries before this
			pb.eventList = pb.eventList[i:]
			return
		}
	}
}

func (p *prefetcher) init() {
	p.prefetchedDataMap = new(sync.Map)
	log.Debug("initialized prefetcher")
}

/*
//-- DEAD CODE -- REMOVE ME LATER --
func (p *prefetcher) getNodePositions_new(epoch *Epoch, id model.NodeId) (*startEndEvents, error) {
	// it's better for the database if we serialize this
	// prefetchDBLock.Lock()
	// defer prefetchDBLock.Unlock()
	var timeStart, timeEnd *model.Event

	// first, figure out the time of the entry at or before t
	if r := model.DB.Limit(1).Find(&timeStart, "dataset_name = ? and node = ? and time <= ?", dataset.DatasetName, id, epoch.prev).Order("time DESC"); r.Error != nil {
		log.Debug("DB fetch failed: ", r.Error)
		return nil, r.Error
	}
	// first, figure out the time of the entry at or before t
	if r := model.DB.Limit(1).Find(&timeEnd, "dataset_name = ? and node = ? and time >= ?", dataset.DatasetName, id, epoch.now).Order("time"); r.Error != nil {
		log.Debug("DB fetch failed: ", r.Error)
		return nil, r.Error
	}
	return &startEndEvents{start: timeStart, end: timeEnd}, nil
}
*/

// figure out the start and end events for the given node and epoch.
// the start event is defined as either occuring at the start of the epoch
// or the one just before the epoch starts.  The end event is defined as the
// one that occurs at the end of the epoch or the one just after the epoch ends.
func (p *prefetcher) getNodePositions(epoch *Epoch, id model.NodeId) (*startEndEvents, error) {

	// check whether we have started prefetching for this nodeID
	var b *prefetcherBin
	e, ok := p.prefetchedDataMap.Load(id)
	if !ok {
		// this is the first time we're attempting to get data for this node, so
		// let's allocate a bin for it
		b = new(prefetcherBin)
		b.init()
		p.prefetchedDataMap.Store(id, b)
	} else {
		b = e.(*prefetcherBin)
	}

	if b.done {
		return nil, nil // signals nothing left for this bin
	}

	// iterate thru this node's events until we find the event that is at the
	// start of the epoch or (if there is no exact match) the one before that
	// log.Infof("prefetching for node %v", id)
	var bounds startEndEvents

	idx := 0

beforeLoop:
	for {

		// do we need to fetch more events?
		l := len(b.eventList)
		if l <= idx {
			var fetchFrom float64
			if l == 0 {
				fetchFrom = epoch.prev
			} else {
				fetchFrom = b.eventList[l-1].Time
			}
			//log.Debugf("need more events for node %v (#events=%v; idx=%v), fetching more starting from time %f; epoch starts at %f", id, l, idx, fetchFrom, epoch.prev)
			isMore, err := b.fetchMoreEvents(id, fetchFrom)
			//log.Infof("node %v feaches from %v, and results is %v", id, fetchFrom, isMore)
			if err != nil {
				log.Fatal(err)
			}
			if !isMore {
				b.done = true
				return nil, nil
			}
		}

		if b.eventList[idx].Time == epoch.prev {
			// we have an exact match
			bounds.start = b.eventList[idx]
			break beforeLoop
		} else if b.eventList[idx].Time < epoch.prev {
			// we need to keep going
			idx += 1
		} else if b.eventList[idx].Time > epoch.prev {
			// this is the weird case where we need to look back one
			if b.eventList[idx-1].Time < epoch.prev {
				bounds.start = b.eventList[idx-1]
			} else {
				panic("logic error")
			}
			break beforeLoop
		} else {
			panic("programming error")

		}
	}
	//log.Debugf("found start bound (%f) for node %v for epoch [%f,%f]", bounds.start.Time, id, epoch.prev, epoch.now)

	// lets get rid of the events before bounds.start
	b.cullEventsBeforeTime(bounds.start.Time)
	if b.eventList[0] != bounds.start { // TODO: we can remove this sanity check later
		panic("programming error")
	}

	// let's do the similar thing for the end time.  Here, we're trying to find
	// the event whose time is greater than or equal to epoch.now.  Note that we
	// start from idx=1 since the 0th entry is bounds.start and
	// bounds.start <> bounds.end.
	idx = 1

afterLoop:
	for {
		// do we need to fetch more events?
		l := len(b.eventList)
		if l <= idx {
			fetchFrom := b.eventList[l-1].Time
			isMore, err := b.fetchMoreEvents(id, fetchFrom)
			if err != nil {
				log.Fatal(err)
			}
			if !isMore {
				b.done = true
				return nil, nil
			}
		}
		//log.Infof("event list: %v, epoch %v", b.eventList[len(b.eventList)-1].Time, epoch.now)
		if b.eventList[idx].Time >= epoch.now {
			bounds.end = b.eventList[idx]
			break afterLoop
		} else {
			idx += 1
		}
	}

	//log.Debugf("found end bound (%f) for node %v for epoch [%f,%f]", bounds.end.Time, id, epoch.prev, epoch.now)

	return &bounds, nil
}
