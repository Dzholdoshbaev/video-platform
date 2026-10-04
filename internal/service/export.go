package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"reflect"
)

type Exporter interface {
	Export(ctx context.Context, data any) ([]byte, error)
}

type JSONExporter struct{}

func (JSONExporter) Export(ctx context.Context, data any) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON export: %w", err)
	}
	return result, nil
}

type CSVExporter struct{}

func (CSVExporter) Export(ctx context.Context, data any) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("CSV exporter expects a slice")
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	for i := 0; i < value.Len(); i++ {
		if err := writer.Write([]string{fmt.Sprint(value.Index(i).Interface())}); err != nil {
			return nil, fmt.Errorf("write CSV row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV export: %w", err)
	}
	return buffer.Bytes(), nil
}

type ExportService struct{ exporter Exporter }

func NewExportService(exporter Exporter) *ExportService { return &ExportService{exporter: exporter} }

// Export demonstrates runtime polymorphism: JSONExporter and CSVExporter are interchangeable.
func (s *ExportService) Export(ctx context.Context, data any) ([]byte, error) {
	return s.exporter.Export(ctx, data)
}
