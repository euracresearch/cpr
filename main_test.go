package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TODO: missing tests for raise and run

var (
	mockPGNumJSON    = `{"pool":"cephfs_data","pool_id":22,"pg_num":1024}`
	mockPGPNumJSON   = `{"pool":"cephfs_data","pool_id":22,"pgp_num":256}`
	mockHealthOKJSON = `{
    "checks": {},
    "status": "HEALTH_OK"
}`
	mockHealthErrJSON = `{
	    "checks": {
	        "OSD_BACKFILLFULL": {
	            "severity": "HEALTH_WARN",
	            "summary": {
	                "message": "1 backfillfull osd(s)"
	            }
	        },
	        "OSD_FULL": {
	            "severity": "HEALTH_ERR",
	            "summary": {
	                "message": "11 full osd(s)"
	            }
	        },
	        "SMALLER_PGP_NUM": {
	            "severity": "HEALTH_WARN",
	            "summary": {
	                "message": "1 pools have pg_num > pgp_num"
	            }
	        }
	    },
	    "status": "HEALTH_ERR"
}`
	mockHealthWarnButOKJSON = `{
	    "checks": {
	        "OSD_BACKFILLFULL": {
	            "severity": "HEALTH_WARN",
	            "summary": {
	                "message": "1 backfillfull osd(s)"
	            }
	        },
	        "OSD_NEARFULL": {
	            "severity": "HEALTH_WARN",
	            "summary": {
	                "message": "21 nearfull osd(s)"
	            }
	        },
	        "SMALLER_PGP_NUM": {
	            "severity": "HEALTH_WARN",
	            "summary": {
	                "message": "1 pools have pg_num > pgp_num"
	            }
	        }
	    },
	    "status": "HEALTH_WARN"
}`

	mockHealthAvailWarnJSON = `{
    "checks": {
        "PG_AVAILABILITY": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "Reduced data availability: 42 pgs inactive, 42 pgs down"
            }
        },
        "SMALLER_PGP_NUM": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "1 pools have pg_num > pgp_num"
            }
        }
    },
    "status": "HEALTH_WARN"
}`
	mockHealhReqWarnJSON = `{
    "checks": {
        "REQUEST_SLOW": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "49 slow requests are blocked > 32 sec"
            }
        },
        "SMALLER_PGP_NUM": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "1 pools have pg_num > pgp_num"
            }
        }
    },
    "status": "HEALTH_WARN"
}`
	mockHealthDegWarnJSON = `{
    "checks": {
        "PG_DEGRADED": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "Degraded data redundancy: 755285/432195232 objects degraded (0.175%), 68 pgs degraded, 68 pgs undersized"
            }
        },
        "SMALLER_PGP_NUM": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "1 pools have pg_num > pgp_num"
            }
        }
    },
    "status": "HEALTH_WARN"
}`
	mockHealthMisWarnJSON = `{
    "checks": {
        "OBJECT_MISPLACED": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "249814/435347950 objects misplaced (0.057%)"
            }
        },
        "SMALLER_PGP_NUM": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "1 pools have pg_num > pgp_num"
            }
        }
    },
    "status": "HEALTH_WARN"
}`
	mockHealthWarnAllJSON = `{
    "checks": {
        "PG_AVAILABILITY": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "Reduced data availability: 42 pgs inactive, 42 pgs down"
            }
        },
        "PG_DEGRADED": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "Degraded data redundancy: 755285/432195232 objects degraded (0.175%), 68 pgs degraded, 68 pgs undersized"
            }
        },
        "REQUEST_SLOW": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "49 slow requests are blocked > 32 sec"
            }
        },
        "OBJECT_MISPLACED": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "249814/435347950 objects misplaced (0.057%)"
            }
        },
        "SMALLER_PGP_NUM": {
            "severity": "HEALTH_WARN",
            "summary": {
                "message": "1 pools have pg_num > pgp_num"
            }
        }
    },
    "status": "HEALTH_WARN"
}`
)

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestIsPowerOfTwo(t *testing.T) {
	testCases := []struct {
		number int64
		want   bool
	}{
		{1, true},
		{2, true},
		{3, false},
		{0, false},
		{7, false},
		{10, false},
		{20, false},
		{1024, true},
		{-2, false},
		{-32, false},
	}

	for _, tc := range testCases {
		if got := powerOfTwo(tc.number); got != tc.want {
			t.Errorf("Number is %d: got %t, want %t", tc.number, got, tc.want)
		}
	}
}

func TestHealthy(t *testing.T) {
	testCases := []struct {
		input string
		name  string
		want  bool
	}{
		{mockHealthOKJSON, "OK", true},
		{mockHealthWarnButOKJSON, "WarnButOK", true},
		{mockHealthErrJSON, "ERR", false},
		{mockHealthAvailWarnJSON, "AVAILABILITY", false},
		{mockHealhReqWarnJSON, "SLOW_REQUESTS", false},
		{mockHealthDegWarnJSON, "DEGRADED", false},
		{mockHealthMisWarnJSON, "MISPLACED", false},
		{mockHealthWarnAllJSON, "ALL", false},
	}

	for _, tc := range testCases {
		if got := healthy([]byte(tc.input)); got != tc.want {
			t.Errorf("%s: got %t, want %t", tc.name, got, tc.want)
		}
	}
}

func TestGet(t *testing.T) {
	execCommand = mockExecCommand

	testCases := []struct {
		pgType string
		want   int64
	}{
		{"pg_num", 1024},
		{"pgp_num", 256},
	}

	for _, tc := range testCases {
		got, err := get("testPool", tc.pgType)
		if err != nil {
			t.Errorf("Got error: %v\n", err)
		}
		if got != tc.want {
			t.Errorf("got %d, want %d", got, tc.want)
		}

	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	// A command looks always like "ceph subcommand arg1 argN".
	// So the real command is the subcommand to check on.
	cmd, args := args[1], args[2:]
	switch cmd {
	case "health":
		fmt.Println(mockHealthOKJSON)
		os.Exit(0)
	case "osd":
		switch args[1] {
		case "get":
			switch args[3] {
			case "pg_num":
				fmt.Println(mockPGNumJSON)
				os.Exit(0)
			case "pgp_num":
				fmt.Println(mockPGPNumJSON)
				os.Exit(0)
			default:
				fmt.Println("invalid")
				os.Exit(22)
			}
		case "set":
		}
	}

}
