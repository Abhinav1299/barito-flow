package flow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BaritoLog/barito-flow/mock"
	pb "github.com/BaritoLog/barito-flow/proto"
	. "github.com/BaritoLog/go-boilerplate/testkit"
	"github.com/BaritoLog/go-boilerplate/timekit"
	"github.com/golang/mock/gomock"
)

func TestProducerService_Produce_OnLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	limiter := NewDummyRateLimiter()
	limiter.Expect_IsHitLimit_AlwaysTrue()

	srv := &producerService{
		limiter: limiter,
	}

	_, err := srv.Produce(nil, pb.SampleTimberProto())
	FatalIfWrongGrpcError(t, onLimitExceededGrpc(), err)
}

func TestProducerService_ProduceBatch_OnLimitExceeded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	limiter := NewDummyRateLimiter()
	limiter.Expect_IsHitLimit_AlwaysTrue()

	srv := &producerService{
		limiter: limiter,
	}

	_, err := srv.ProduceBatch(nil, pb.SampleTimberCollectionProto())
	FatalIfWrongGrpcError(t, onLimitExceededGrpc(), err)
}

func TestProducerService_Produce_OnCreateTopicError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	admin := mock.NewMockKafkaAdmin(ctrl)
	admin.EXPECT().Exist(gomock.Any()).Return(false)
	admin.EXPECT().CreateTopic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("create-topic-error"))

	producer := mock.NewMockSyncProducer(ctrl)
	limiter := NewDummyRateLimiter()

	srv := &producerService{
		producer:    producer,
		topicSuffix: "_logs",
		admin:       admin,
		limiter:     limiter,
	}

	_, err := srv.Produce(nil, pb.SampleTimberProto())
	FatalIfWrongGrpcError(t, onCreateTopicErrorGrpc(fmt.Errorf("")), err)
}

func TestProducerService_Produce_OnStoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	admin := mock.NewMockKafkaAdmin(ctrl)
	admin.EXPECT().Exist(gomock.Any()).Return(true)

	producer := mock.NewMockSyncProducer(ctrl)
	producer.EXPECT().SendMessage(gomock.Any()).
		Return(int32(0), int64(0), fmt.Errorf("some-error"))

	limiter := NewDummyRateLimiter()

	srv := &producerService{
		producer:    producer,
		topicSuffix: "_logs",
		admin:       admin,
		limiter:     limiter,
	}

	_, err := srv.Produce(nil, pb.SampleTimberProto())
	FatalIfWrongGrpcError(t, onStoreErrorGrpc(fmt.Errorf("")), err)
}

func TestProducerService_Produce_OnSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	admin := mock.NewMockKafkaAdmin(ctrl)
	admin.EXPECT().Exist(gomock.Any()).Return(false)
	admin.EXPECT().CreateTopic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	admin.EXPECT().AddTopic(gomock.Any())

	producer := mock.NewMockSyncProducer(ctrl)
	producer.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	limiter := NewDummyRateLimiter()

	srv := &producerService{
		producer:    producer,
		topicSuffix: "_logs",
		admin:       admin,
		limiter:     limiter,
	}

	resp, err := srv.Produce(nil, pb.SampleTimberProto())
	FatalIfError(t, err)
	FatalIf(t, resp.GetTopic() != "some_topic_logs", "wrong result.Topic")
}

func TestProducerService_ProduceBatch_OnSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	admin := mock.NewMockKafkaAdmin(ctrl)
	admin.EXPECT().Exist(gomock.Any()).Return(false)
	admin.EXPECT().CreateTopic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	admin.EXPECT().AddTopic(gomock.Any())
	admin.EXPECT().Exist(gomock.Any()).Return(false)
	admin.EXPECT().CreateTopic(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	admin.EXPECT().AddTopic(gomock.Any())

	producer := mock.NewMockSyncProducer(ctrl)
	producer.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	limiter := NewDummyRateLimiter()

	srv := &producerService{
		producer:    producer,
		topicSuffix: "_logs",
		admin:       admin,
		limiter:     limiter,
	}

	resp, err := srv.ProduceBatch(nil, pb.SampleTimberCollectionProto())
	FatalIfError(t, err)
	FatalIf(t, resp.GetTopic() != "some_topic_logs", "wrong result.Topic")
}

func TestProducerService_Start_ErrorMakeSyncProducer(t *testing.T) {
	factory := NewDummyKafkaFactory()
	factory.Expect_MakeSyncProducerFunc_AlwaysError("some-error")

	producerParams := map[string]interface{}{
		"factory":                factory,
		"grpcAddr":               "grpc",
		"restAddr":               "rest",
		"rateLimitResetInterval": 1,
		"topicSuffix":            "_logs",
		"kafkaMaxRetry":          1,
		"kafkaRetryInterval":     10,
		"newEventTopic":          "new_topic_events",
	}

	service := NewProducerService(producerParams)
	err := service.Start()

	FatalIfWrongError(t, err, "Make sync producer failed: Error connecting to kafka, retry limit reached")
}

func TestProducerService_Start_ErrorMakeKafkaAdmin(t *testing.T) {
	factory := NewDummyKafkaFactory()
	factory.Expect_MakeKafkaAdmin_AlwaysError("some-error")

	producerParams := map[string]interface{}{
		"factory":                factory,
		"grpcAddr":               "grpc",
		"restAddr":               "rest",
		"rateLimitResetInterval": 1,
		"topicSuffix":            "_logs",
		"kafkaMaxRetry":          1,
		"kafkaRetryInterval":     10,
		"newEventTopic":          "new_topic_events",
	}

	service := NewProducerService(producerParams)
	err := service.Start()

	FatalIfWrongError(t, err, "Make kafka admin failed: Error connecting to kafka, retry limit reached")
}

func TestProducerService_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	factory := NewDummyKafkaFactory()
	factory.Expect_MakeKafkaAdmin_ProducerServiceSuccess(ctrl, []string{})

	service := &producerService{
		factory:       factory,
		grpcAddr:      ":24400",
		topicSuffix:   "_logs",
		newEventTopic: "new_topic_event",
	}

	var err error
	go func() {
		err = service.Start()
	}()
	defer service.Close()

	FatalIfError(t, err)

	timekit.Sleep("1ms")
	FatalIf(t, !service.limiter.IsStart(), "rate limiter must be start")
}

func FatalIfWrongGrpcError(t *testing.T, expected error, actual error) {
	expFields := strings.Fields(expected.Error())[:5]
	expStr := strings.Join(expFields, " ")

	actFields := strings.Fields(actual.Error())[:5]
	actStr := strings.Join(actFields, " ")

	if expStr != actStr {
		t.Errorf("expected gRPC response code %v, received %v.", expFields[4], actFields[4])
	}
}
