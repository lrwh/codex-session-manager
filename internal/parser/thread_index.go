package parser

import (
	"bufio"
	"encoding/json"
	"os"
)

type ThreadIndexSummary struct {
	ThreadName string
	UpdatedAt  string
}

type threadIndexRecord struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
	UpdatedAt  string `json:"updated_at"`
}

func ParseThreadIndexFile(path string) (map[string]ThreadIndexSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ThreadIndexSummary{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 16*1024*1024)

	summaries := make(map[string]ThreadIndexSummary)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record threadIndexRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		if record.ID == "" || record.ThreadName == "" {
			continue
		}

		existing := summaries[record.ID]
		if existing.ThreadName == "" || record.UpdatedAt >= existing.UpdatedAt {
			summaries[record.ID] = ThreadIndexSummary{
				ThreadName: record.ThreadName,
				UpdatedAt:  record.UpdatedAt,
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}

