// Copyright 2018 Eurac Research. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// cpr - ceph placement group raiser
//
// Usage:
//
//      cpr -pool my_fancy_pool -target 512
//      cpr -pool my_fancy_pool -target 1024 -delta 5
//      cpr -pool my_fancy_pool -target 256 -verbose
//
// "cpr" raises ceph placement groups of a given pool step
// by step. It will first raise the 'pg_num' of the pool to
// the given target and then waiting 30 seconds before
// proceeding with the 'pgp_num'.
//
// Before each raise it will be checked if the cluster is
// in a healthy state for raising placement groups, if not
// it will wait 10 seconds before retrying. After the raise
// it will wait additional 40 seconds for Ceph to recognise
// the change.
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	// execCommand is a variable here so that it may be overridden in
	// tests.
	execCommand = exec.Command

	flagPool    = flag.String("pool", "", "ceph pool name")
	flagDelta   = flag.Int64("delta", 10, "")
	flagTarget  = flag.Int64("target", 0, "target PG number")
	flagVerbose = flag.Bool("verbose", false, "verbose output")
)

func main() {
	log.SetPrefix("cpr: ")
	log.SetFlags(0)
	flag.Parse()

	if *flagPool == "" {
		fmt.Fprintf(os.Stderr, "A pool name must be provided.\n\n")
		flag.Usage()
		os.Exit(2)
	}

	if !powerOfTwo(*flagTarget) {
		fmt.Fprintf(os.Stderr, "Target PG number must be greater then 0 and a power of 2\n\n")
		flag.Usage()
		os.Exit(2)
	}

	log.Printf("Starting in 10s to raise 'pg_num' of %q to %d.\n", *flagPool, *flagTarget)
	run("pg_num")

	log.Printf("Waiting 30s then continuing raising 'pgp_num' of %q to %d.\n", *flagPool, *flagTarget)
	time.Sleep(30 * time.Second)

	run("pgp_num")
}

// run kicks off the process for raising the placement groups.
func run(pgType string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// On ^C, or SIGTERM handle exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	done := make(chan bool, 1)
	for {
		select {
		case <-ticker.C:
			err := raise(*flagPool, pgType, *flagTarget, *flagDelta, done)
			if err != nil {
				log.Fatalf("raise: Error in raising %s of %s: %v", *flagPool, pgType, err)
			}
		case <-done:
			log.Printf("DONE: %s of %q is now %d.\n", pgType, *flagPool, *flagTarget)
			return
		case <-c:
			os.Exit(1)
		}
	}
}

// powerOfTwo reports whether n is a power of two.
func powerOfTwo(n int64) bool {
	return n != 0 && n&(n-1) == 0
}

// Health holds the cluster health status.
type Health struct {
	Checks struct {
		Availability struct {
			Status string `json:"severity"`
		} `json:"PG_AVAILABILITY"`
		Degraded struct {
			Status string `json:"severity"`
		} `json:"PG_DEGRADED"`
		SlowRequest struct {
			Status string `json:"severity"`
		} `json:"REQUEST_SLOW"`
		Misplaced struct {
			Status string `json:"severity"`
		} `json:"OBJECT_MISPLACED"`
	}
	Status string
}

// healthy reports if the cluster is in healthy state for raising placement
// groups.
func healthy(j []byte) bool {
	var h Health
	if err := json.Unmarshal(j, &h); err != nil {
		log.Fatalf("health: Could not unmarshal json: %v. stopping.", err)
	}

	if *flagVerbose {
		log.Printf("health: %v\n", h)
	}

	// cluster overall health
	switch h.Status {
	case "HEALTH_OK":
		return true
	case "HEALTH_ERR":
		return false
	}

	c := h.Checks
	switch {
	case c.Availability.Status == "HEALTH_WARN":
		return false
	case c.Degraded.Status == "HEALTH_WARN":
		return false
	case c.SlowRequest.Status == "HEALTH_WARN":
		return false
	case c.Misplaced.Status == "HEALTH_WARN":
		return false
	}

	return true
}

// get gets the current pg_num or pgp_num according to the passed pgType of
// the given pool.
func get(pool, pgType string) (int64, error) {
	b, err := runCmd("ceph", "osd", "pool", "get", pool, pgType, "-f", "json")
	if err != nil {
		return -1, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return -1, err
	}

	v, ok := m[pgType]
	if !ok {
		return -1, fmt.Errorf("Error in getting %s of %s", pgType, pool)
	}

	return int64(v.(float64)), nil
}

// raise raises the placement group of the given pool and pg type by the given
// delta.
func raise(pool, pgType string, target, delta int64, done chan bool) error {
	h, err := runCmd("ceph", "health", "-f", "json")
	if err != nil {
		return fmt.Errorf("health: Error reading health status: %v", err)
	}

	if !healthy(h) {
		if *flagVerbose {
			log.Println("Cluster is not healthy. Retrying.")
		}
		return nil
	}

	cPg, err := get(pool, pgType)
	if err != nil {
		return err
	}

	if cPg >= target {
		done <- true
		return nil
	}

	nPg := cPg + delta
	if nPg > target {
		nPg = target
	}

	_, err = runCmd("ceph", "osd", "pool", "set", pool, pgType, strconv.FormatInt(nPg, 10))
	if err != nil {
		return err
	}

	log.Printf("Raising %s of %q from %d to %d (target=%d)\n", pgType, pool, cPg, nPg, target)
	log.Printf("Waiting 40s for Ceph to recognize the change before continuing.")

	time.Sleep(40 * time.Second)

	return nil
}

// runCmd runs the command line, returning its output. If the command cannot
// be run or does return a bad exit code, it will return an error.
func runCmd(command string, args ...string) ([]byte, error) {
	if *flagVerbose {
		log.Printf("runCmd: %s %s", command, strings.Join(args, " "))
	}

	out, err := execCommand(command, args...).CombinedOutput()
	if err != nil {
		log.Printf("runCmd: %s", out)
		return nil, err
	}

	return out, nil
}
