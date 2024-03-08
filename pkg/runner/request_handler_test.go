package runner

import (
	"context"
	"dcard-backend-2024/pkg/inmem"
	"dcard-backend-2024/pkg/model"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRunner_IsRunning(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan map[string]chan interface{}
		Store        model.InMemoryStore
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Test IsRunning",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: make(map[string]chan interface{}),
				Store:        nil,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRunner(tt.fields.Store)
			go r.Start()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			for {
				if r.IsRunning() {
					break
				}
				select {
				case <-ctx.Done():
					t.Errorf("Runner.IsRunning() = %v, want %v", r.IsRunning(), tt.want)
				default:
				}
			}
		})
	}
}

func TestRunner_handleCreateBatchAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan map[string]chan interface{}
		Store        model.InMemoryStore
	}
	type args struct {
		req *CreateBatchAdRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test handleCreateBatchAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: make(map[string]chan interface{}),
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &CreateBatchAdRequest{
					Request: Request{RequestID: "test"},
					Ads: []*model.Ad{
						{
							ID:       uuid.New(),
							Title:    "test",
							Content:  "test",
							StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
							EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
							AgeStart: 18,
							AgeEnd:   65,
							Gender:   []string{"F", "M"},
							Country:  []string{"TW"},
							Platform: []string{"ios"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}
			tt.fields.ResponseChan[tt.args.req.RequestID] = make(chan interface{})
			go r.handleCreateBatchAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan[tt.args.req.RequestID]:
				if resp, ok := resp.(*CreateAdResponse); ok {
					if resp.Err != nil {
						t.Errorf("Runner.handleCreateBatchAdRequest() = %v, want %v", resp.Err, nil)
					}
					assert.Equal(t, resp.AdID, "")
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Runner.handleCreateBatchAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestRunner_handleCreateAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan map[string]chan interface{}
		Store        model.InMemoryStore
	}
	type args struct {
		req *CreateAdRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test handleCreateAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: make(map[string]chan interface{}),
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &CreateAdRequest{
					Request: Request{RequestID: "test"},
					Ad: &model.Ad{
						ID:       uuid.New(),
						Title:    "test",
						Content:  "test",
						StartAt:  model.CustomTime(time.Now().Add(-1 * time.Hour * 24)),
						EndAt:    model.CustomTime(time.Now().Add(1 * time.Hour * 24)),
						AgeStart: 18,
						AgeEnd:   65,
						Gender:   []string{"F", "M"},
						Country:  []string{"TW"},
						Platform: []string{"ios"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}
			tt.fields.ResponseChan[tt.args.req.RequestID] = make(chan interface{})
			go r.handleCreateAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan[tt.args.req.RequestID]:
				if resp, ok := resp.(*CreateAdResponse); ok {
					if resp.Err != nil {
						t.Errorf("Runner.handleCreateAdRequest() = %v, want %v", resp.Err, nil)
					}
					assert.Equal(t, resp.AdID, tt.args.req.Ad.ID.String())
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Runner.handleCreateAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

func TestRunner_handleGetAdRequest(t *testing.T) {
	type fields struct {
		RequestChan  chan interface{}
		ResponseChan map[string]chan interface{}
		Store        model.InMemoryStore
	}
	type args struct {
		req *GetAdRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test handleGetAdRequest",
			fields: fields{
				RequestChan:  make(chan interface{}),
				ResponseChan: make(map[string]chan interface{}),
				Store:        inmem.NewInMemoryStore(),
			},
			args: args{
				req: &GetAdRequest{
					Request: Request{RequestID: "test"},
					GetAdRequest: &model.GetAdRequest{
						Age:     18,
						Country: "TW",
						Limit:   10,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				RequestChan:  tt.fields.RequestChan,
				ResponseChan: tt.fields.ResponseChan,
				Store:        tt.fields.Store,
			}

			tt.fields.ResponseChan[tt.args.req.RequestID] = make(chan interface{})
			go r.handleGetAdRequest(tt.args.req)
			select {
			case resp := <-tt.fields.ResponseChan[tt.args.req.RequestID]:
				if resp, ok := resp.(*GetAdResponse); ok {
					assert.ErrorIs(t, resp.Err, inmem.ErrNoAdsFound)
				}
			case <-time.After(3 * time.Second):
				t.Errorf("Runner.handleGetAdRequest() = %v, want %v", nil, nil)
			}
		})
	}
}

			}

func TestNewRunner(t *testing.T) {
	type args struct {
		store model.InMemoryStore
	}
	tests := []struct {
		name string
		args args
		want *Runner
	}{
		{
			name: "Test NewRunner",
			args: args{
				store: inmem.NewInMemoryStore(),
			},
			want: &Runner{
				RequestChan:  make(chan interface{}),
				ResponseChan: make(map[string]chan interface{}),
				Store:        inmem.NewInMemoryStore(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Store = tt.args.store
			if got := NewRunner(tt.args.store); !reflect.DeepEqual(got.Store, tt.want.Store) {
				t.Errorf("NewRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}
