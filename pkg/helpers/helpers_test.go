package helpers

import (
	"testing"
)

func TestConvertStartStopToSliceIndexs(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}

	// converts start into start - 1, unless start = 0
	indexStart, indexStop := ConvertStartStopToSliceIndexes(1, 10, int32(len(slice)))
	expectStart, expectStop := int32(0), int32(10)
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}
	indexStart, indexStop = ConvertStartStopToSliceIndexes(0, 10, int32(len(slice)))
	expectStart, expectStop = int32(0), int32(10)
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}

	// returns stop no longer than length of slice
	indexStart, indexStop = ConvertStartStopToSliceIndexes(1, 30, int32(len(slice)))
	expectStart, expectStop = int32(0), int32(25)
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}

	// converts negative start to 0
	indexStart, indexStop = ConvertStartStopToSliceIndexes(-1, 10, int32(len(slice)))
	expectStart, expectStop = int32(0), int32(10)
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}

	// returns all for 0, 0 or 1, 0
	indexStart, indexStop = ConvertStartStopToSliceIndexes(1, 0, int32(len(slice)))
	expectStart, expectStop = int32(0), int32(0)
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}
	indexStart, indexStop = ConvertStartStopToSliceIndexes(0, 0, int32(len(slice)))
	if indexStart != expectStart || indexStop != expectStop {
		t.Errorf("Expected: %d, %d, got: %d, %d", expectStart, expectStop, indexStart, indexStop)
	}
}
