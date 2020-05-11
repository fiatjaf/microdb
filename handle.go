package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func handleBucketAccess(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]

	path := strings.Split(r.URL.Path, "/")
	path = path[2:]

	log := log.With().Str("bucket", bucket).
		Str("path", strings.Join(path, "/")).
		Str("method", r.Method).
		Logger()

	if r.Method == "GET" {
		log.Debug().Msg("bucket access")

		var value types.JSONText
		err := pg.Get(&value, `
SELECT data #> $2
FROM data
WHERE bucket = $1
  AND bucket_credits($1) > 0
        `, bucket, pq.Array(path))
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

		json.NewEncoder(w).Encode(value)
	} else {
		var accessKey string
		auth := strings.Split(r.Header.Get("Authorization"), " ")
		if len(auth) > 0 {
			accessKey = auth[len(auth)-1]
		}

		defer r.Body.Close()
		data, _ := ioutil.ReadAll(r.Body)

		log.Debug().Str("data", string(data)).Str("key", accessKey).
			Msg("bucket access")

		params := []interface{}{bucket, accessKey}

		var update string
		switch r.Method {
		case "PUT":
			// update object at this point
			if len(path) == 0 {
				update = "$3"
				params = append(params, types.JSONText(data))
			} else {
				update = `jsonb_set(data, $3, $4, true)`
				params = append(params, pq.Array(path))
				params = append(params, types.JSONText(data))
			}
		case "POST":
			// append to array at this point
			update = `jsonb_insert(
                  jsonb_set(data, $3, coalesce(data #> $3, '[]'::jsonb)),
                  $4, $5, true
                )`
			params = append(params, pq.Array(path))
			params = append(params, pq.Array(append(path, "-1")))
			params = append(params, types.JSONText(data))
		case "PATCH":
			// merge arrays or objects at path (must necessarily exist beforehand)
			update = `jsonb_set(data, $3, data #> $3 || $4)`
			params = append(params, pq.Array(path))
			params = append(params, types.JSONText(data))
		case "DELETE":
			if len(path) == 0 {
				update = "'{}'::jsonb"
			} else {
				update = `jsonb_strip_nulls(jsonb_set(data, $3, 'null'::jsonb))`
				params = append(params, pq.Array(path))
			}
		}

		_, err := pg.Exec(`
UPDATE data
  SET data = `+update+`
WHERE bucket = $1
  AND (public_write OR key = $2)
  AND bucket_credits($1) > 0
    `, params...)
		if err != nil {
			log.Debug().Err(err).Msg("update failed")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(ErrorResponse{err.Error()})
			return
		}

		w.WriteHeader(200)
	}
}
