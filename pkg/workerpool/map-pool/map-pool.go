package map_pool

import (
	"github.com/rs/zerolog/log"
	"sync"
)

// JobWithResult represents a job function that takes an input of type T and returns a result of type U and an error.
type JobWithResult[U any] func() (U, error)

type Pool[U any] struct {
	workerCount int
	jobs        chan JobWithResult[U]
	results     chan U
	wg          sync.WaitGroup
}

func New[U any](workerCount int) *Pool[U] {
	return &Pool[U]{
		workerCount: workerCount,
		jobs:        make(chan JobWithResult[U]),
		results:     make(chan U, workerCount),
	}
}

func (p *Pool[U]) worker(id int) {
	defer func() {
		p.wg.Done()
	}()
	for {
		job, ok := <-p.jobs
		if !ok {
			log.Info().Int("worker", id).Msg("Worker finished")
			return
		}
		jobResult, err := job()
		if err != nil {
			log.Error().Err(err).Int("worker", id).Msg("Error executing job")
			continue
		}
		p.results <- jobResult
	}
}

func (p *Pool[U]) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *Pool[U]) AddJob(job JobWithResult[U]) {
	p.jobs <- job
}

func (p *Pool[U]) Close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

func (p *Pool[U]) Results() <-chan U {
	return p.results
}
