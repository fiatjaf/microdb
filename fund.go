package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fiatjaf/go-lnurl"
	"github.com/gorilla/mux"
	"github.com/lucsky/cuid"
)

func fundBucket(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]
	log.Debug().Str("bucket", bucket).Msg("bucket fund")

	amount, _ := strconv.ParseInt(r.URL.Query().Get("amount"), 10, 64)
	if amount == 0 {
		// return params
		json.NewEncoder(w).Encode(lnurl.LNURLPayResponse1{
			Tag:             "payRequest",
			Callback:        s.ServiceURL + "/fund/" + bucket,
			MinSendable:     1000,
			MaxSendable:     100000000,
			EncodedMetadata: string(makeMetadata(bucket)),
		})
	} else {
		// return invoice
		h := sha256.Sum256(makeMetadata(bucket))
		label := "lndb." + bucket + "." + cuid.Slug()

		res, err := spark.Call("invoicewithdescriptionhash",
			amount,
			label,
			hex.EncodeToString(h[:]),
		)

		if err != nil {
			log.Error().Err(err).Str("label", label).Int64("amount", amount).
				Msg("failed to create invoice")
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("failed to create invoice"))
			return
		}

		json.NewEncoder(w).Encode(lnurl.LNURLPayResponse2{
			PR:         res.Get("bolt11").String(),
			Routes:     make([][]lnurl.RouteInfo, 0),
			Disposable: lnurl.FALSE,
		})
	}
}

func makeMetadata(bucket string) []byte {
	j, _ := json.Marshal([][]string{
		[]string{"text/plain", fmt.Sprintf("Fund bucket %s.", bucket)},
	})
	return j
}
