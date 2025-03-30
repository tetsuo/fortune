package frontend

import (
	"net/http"

	"cloud.google.com/go/errorreporting"
	"github.com/tetsuo/fortune/internal/database"
	"go.uber.org/zap"
)

type Server struct {
	db  *database.DB
	log *zap.SugaredLogger
	er  *errorreporting.Client
}

func NewServer(cfg Config, db *database.DB, er *errorreporting.Client) (*Server, error) {
	return &Server{
		log: zap.S(),
		er:  er,
		db:  db,
	}, nil
}

func (s *Server) Install(handle func(string, http.Handler)) {
	handle("GET /", s.errorHandler(s.serveGET))
	handle("POST /", s.errorHandler(s.servePOST))
	handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}
