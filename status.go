package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Status struct {
	Bucket string `json:"bucket" db:"bucket"`
	Size   int64  `json:"size" db:"size"`
	Funds  int64  `json:"funds" db:"funds"`
}

func getBucketStatus(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]

	var status Status
	err := pg.Get(&status, `
SELECT bucket, pg_column_size(data) AS size, bucket_credits(bucket) AS funds
FROM data WHERE bucket = $1
        `, bucket)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(ErrorResponse{"no bucket " + bucket})
			return
		}

		log.Debug().Err(err).Msg("read failed")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(ErrorResponse{err.Error()})
		return
	}

	json.NewEncoder(w).Encode(status)
}
