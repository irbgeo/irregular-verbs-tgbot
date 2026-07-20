package service

import (
	"math/rand"
	"sort"
	"time"
)

// Service holds all business logic and depends only on repository ports.
type Service struct {
	users    UserRepository
	byBase   map[string]Verb
	byLevel  map[string][]Verb
	allBases []string
	now      func() time.Time
	rng      func(n int) int // returns [0,n)
}

// New builds the in-memory verb catalog and returns a Service.
func New(users UserRepository, verbs []Verb) *Service {
	s := &Service{
		users:   users,
		byBase:  make(map[string]Verb, len(verbs)),
		byLevel: make(map[string][]Verb),
		now:     time.Now,
		rng: func(n int) int {
			if n <= 0 {
				return 0
			}
			return rand.Intn(n)
		},
	}
	for _, v := range verbs {
		s.byBase[v.Base] = v
		s.byLevel[v.Level] = append(s.byLevel[v.Level], v)
	}
	for lvl := range s.byLevel {
		ws := s.byLevel[lvl]
		sort.Slice(ws, func(i, j int) bool { return ws[i].Base < ws[j].Base })
	}
	s.allBases = make([]string, 0, len(verbs))
	for _, lvl := range Levels {
		for _, v := range s.byLevel[lvl] {
			s.allBases = append(s.allBases, v.Base)
		}
	}
	return s
}

func (s *Service) verb(base string) (*Verb, bool) { v, ok := s.byBase[base]; return &v, ok }

func (s *Service) levelWords(level string) []Verb { return s.byLevel[level] }
