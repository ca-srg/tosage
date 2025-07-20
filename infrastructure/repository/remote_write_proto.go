package repository

import (
	"bytes"
	"encoding/binary"
	"math"
)

// encodeWriteRequest manually encodes a WriteRequest into protobuf format
func encodeWriteRequest(metricName string, value float64, labels map[string]string, timestamp int64) ([]byte, error) {
	var buf bytes.Buffer

	// Create labels including __name__
	allLabels := make(map[string]string)
	allLabels["__name__"] = metricName
	for k, v := range labels {
		allLabels[k] = v
	}

	// Field 1: timeseries (repeated)
	timeseriesData := encodeTimeSeries(allLabels, value, timestamp)
	writeFieldWithData(&buf, 1, 2, timeseriesData) // field 1, wire type 2 (length-delimited)

	return buf.Bytes(), nil
}

// encodeTimeSeries encodes a single TimeSeries
func encodeTimeSeries(labels map[string]string, value float64, timestamp int64) []byte {
	var buf bytes.Buffer

	// Field 1: labels (repeated)
	for name, val := range labels {
		labelData := encodeLabel(name, val)
		writeFieldWithData(&buf, 1, 2, labelData)
	}

	// Field 2: samples (repeated)
	sampleData := encodeSample(value, timestamp)
	writeFieldWithData(&buf, 2, 2, sampleData)

	return buf.Bytes()
}

// encodeLabel encodes a single Label
func encodeLabel(name, value string) []byte {
	var buf bytes.Buffer

	// Field 1: name (string)
	writeString(&buf, 1, name)

	// Field 2: value (string)
	writeString(&buf, 2, value)

	return buf.Bytes()
}

// encodeSample encodes a single Sample
func encodeSample(value float64, timestamp int64) []byte {
	var buf bytes.Buffer

	// Field 1: value (double/fixed64)
	writeFixed64(&buf, 1, math.Float64bits(value))

	// Field 2: timestamp (int64/varint)
	writeVarint(&buf, 2, timestamp)

	return buf.Bytes()
}

// writeFieldWithData writes a field number and wire type followed by length-delimited data
func writeFieldWithData(buf *bytes.Buffer, fieldNum int, wireType int, data []byte) {
	key := (fieldNum << 3) | wireType
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(len(data)))
	buf.Write(data)
}

// writeString writes a string field
func writeString(buf *bytes.Buffer, fieldNum int, s string) {
	key := (fieldNum << 3) | 2 // wire type 2 for string
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(len(s)))
	buf.WriteString(s)
}

// writeFixed64 writes a fixed64 field
func writeFixed64(buf *bytes.Buffer, fieldNum int, v uint64) {
	key := (fieldNum << 3) | 1 // wire type 1 for fixed64
	writeRawVarint(buf, uint64(key))
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// writeVarint writes a varint field
func writeVarint(buf *bytes.Buffer, fieldNum int, v int64) {
	key := fieldNum << 3 // wire type 0 for varint (| 0 is redundant)
	writeRawVarint(buf, uint64(key))
	writeRawVarint(buf, uint64(v))
}

// writeRawVarint writes a raw varint value
func writeRawVarint(buf *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}
