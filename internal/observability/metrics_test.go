package observability

import "testing"

func TestRecordQA_NoPanic(t *testing.T) {
	RecordQA("rule_exact")
	RecordQA("test_source")
}
