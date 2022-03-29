package filter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/common/model"
)

func BenchmarkWriteLabels_StringBuilder(b *testing.B) {
	ls := map[model.LabelName]model.LabelValue{
		model.LabelName("label_1"): model.LabelValue("value_1"),
		model.LabelName("label_2"): model.LabelValue("value_2"),
		model.LabelName("label_3"): model.LabelValue("value_3"),
	}

	l := model.LabelSet(ls)
	for i := 0; i < b.N; i++ {
		LabelsString(&l)
	}
}

func BenchmarkWriteLabels_StringBuilderiByte(b *testing.B) {
	ls := map[model.LabelName]model.LabelValue{
		model.LabelName("label_1"): model.LabelValue("value_1"),
		model.LabelName("label_2"): model.LabelValue("value_2"),
		model.LabelName("label_3"): model.LabelValue("value_3"),
	}

	l := model.LabelSet(ls)
	for i := 0; i < b.N; i++ {
		labelsStringByte(&l)
	}
}

func BenchmarkWriteLabels_Sprintf(b *testing.B) {
	ls := map[model.LabelName]model.LabelValue{
		model.LabelName("label_1"): model.LabelValue("value_1"),
		model.LabelName("label_2"): model.LabelValue("value_2"),
		model.LabelName("label_3"): model.LabelValue("value_3"),
	}

	l := model.LabelSet(ls)
	for i := 0; i < b.N; i++ {
		printf(&l)
	}
}

func BenchmarkWriteLabels_Simple(b *testing.B) {
	ls := map[model.LabelName]model.LabelValue{
		model.LabelName("label_1"): model.LabelValue("value_1"),
		model.LabelName("label_2"): model.LabelValue("value_2"),
		model.LabelName("label_3"): model.LabelValue("value_3"),
	}

	l := model.LabelSet(ls)
	for i := 0; i < b.N; i++ {
		simple(&l)
	}
}

func simple(l *model.LabelSet) string {
	lstrs := make([]string, 0, len(*l))
	for l, v := range *l {
		tmp := string(l) + "=\"" + string(v) + "\""
		lstrs = append(lstrs, tmp)
	}

	return "{" + strings.Join(lstrs, ", ") + "}"
}

func printf(labels *model.LabelSet) string {
	var b strings.Builder
	b.WriteString("{")
	for l, v := range *labels {
		fmt.Fprintf(&b, "%s=%q, ", l, v)
	}
	b.WriteString("}")
	s := b.String()
	return s[:b.Len()-2]
}

func labelsStringByte(labels *model.LabelSet) string {
	var b strings.Builder
	b.WriteByte('{')
	for l, v := range *labels {
		b.WriteString(string(l))
		b.WriteByte('=')
		b.WriteByte('"')
		b.WriteString(string(v))
		b.WriteByte('"')
		b.WriteByte(',')
		b.WriteByte(' ')
	}
	b.WriteByte('}')
	s := b.String()
	return s[:b.Len()-2]
}
