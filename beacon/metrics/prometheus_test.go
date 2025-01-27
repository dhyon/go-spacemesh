package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/spacemeshos/go-spacemesh/common/types"
)

func TestBeaconMetrics(t *testing.T) {
	epoch := types.EpochID(10)
	observed := []*BeaconStats{
		{
			Epoch:  epoch,
			Beacon: "canadian",
			Count:  uint64(123),
			Weight: uint64(32100),
		},
		{
			Epoch:  epoch,
			Beacon: "rashers",
			Count:  uint64(321),
			Weight: uint64(12300),
		},
	}
	calculated := &BeaconStats{
		Epoch:  epoch + 1,
		Beacon: "speck",
		Count:  uint64(1),
		Weight: uint64(45678),
	}

	bmc := NewBeaconMetricsCollector(func() ([]*BeaconStats, *BeaconStats) {
		return observed, calculated
	}, nil)

	deviceExpected := `
# HELP spacemesh_beacons_beacon_calculated_weight Weight of the beacon calculated by the node for each epoch
# TYPE spacemesh_beacons_beacon_calculated_weight counter
spacemesh_beacons_beacon_calculated_weight{beacon="speck",epoch="11"} 45678
# HELP spacemesh_beacons_beacon_observed_total Number of beacons collected from blocks for each epoch and value
# TYPE spacemesh_beacons_beacon_observed_total counter
spacemesh_beacons_beacon_observed_total{beacon="canadian",epoch="10"} 123
spacemesh_beacons_beacon_observed_total{beacon="rashers",epoch="10"} 321
# HELP spacemesh_beacons_beacon_observed_weight Weight of beacons collected from blocks for each epoch and value
# TYPE spacemesh_beacons_beacon_observed_weight counter
spacemesh_beacons_beacon_observed_weight{beacon="canadian",epoch="10"} 32100
spacemesh_beacons_beacon_observed_weight{beacon="rashers",epoch="10"} 12300
`
	if err := testutil.CollectAndCompare(bmc, strings.NewReader(deviceExpected)); err != nil {
		t.Error(err)
	}
}
