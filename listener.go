package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/donovanhide/eventsource"
	"github.com/tidwall/gjson"
)

func sparkoListener() {
	req, _ := http.NewRequest("GET", s.SparkURL+"/stream", nil)
	req.Header.Set("X-Access", s.SparkToken)
	stream, err := eventsource.SubscribeWithRequest("", req)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to subscribe to sparko.")
	}

	defer stream.Close()

	go func() {
		for err := range stream.Errors {
			log.Debug().Err(err).Msg("spark stream error")
		}
	}()

	for ev := range stream.Events {
		switch ev.Event() {
		case "invoice_payment":
			data := gjson.Parse(ev.Data())
			label := data.Get("invoice_payment.label").String()
			if !strings.HasPrefix(label, "microdb.") {
				continue
			}
			preimage := data.Get("invoice_payment.preimage").String()
			msat := data.Get("invoice_payment.msat").String()
			log.Debug().Str("label", label).Str("r", preimage).Str("msat", msat).
				Msg("got payment")

			amount, err := strconv.ParseInt(strings.TrimSuffix(msat, "msat"), 10, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse msat on invoice_payment")
				return
			}

			p := strings.Split(label, ".")
			if len(p) != 3 {
				log.Error().Err(err).Msg("failed to parse bucket on invoice_payment")
				return
			}
			bucket := p[1]

			_, err = pg.Exec(`
INSERT INTO funding (time, bucket, msatoshi)
VALUES (now(), $1, $2)
            `, bucket, amount,
			)
			if err != nil {
				log.Error().Err(err).Msg("failed to save invoice_payment")
				return
			}

			pg.Exec(`
INSERT INTO data (bucket, data) VALUES ($1, '{}'::jsonb)
ON CONFLICT (bucket) DO NOTHING
            `, bucket)
		}
	}
}
