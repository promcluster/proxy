package dns

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"
)

func TestProvider(t *testing.T) {
	p := NewProvider("golang", zap.NewExample())
	res, err := p.Resolve(context.TODO(), "dns+qq.com:80")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
}
