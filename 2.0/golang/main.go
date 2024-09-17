package main

import (
	"context"
	pb "externalscaler-sample/externalscaler"
	"fmt"
	"log"
	"net"
	"strconv"

	// "log"
	// "net"

	"time"

	"google.golang.org/grpc"
)

type ExternalScaler struct{}

const METRIC_NAME = "QUEUE"
const METRIC_TARGETSIZE = 2 // TODO change to DEFAULT

const SCALEDOBJECT_KEY_METRIC_TARGETSIZE = "metricTargetSize"

func getCurrentMetric() (metric int) {
	_, minutes, _ := time.Now().Clock()
	metric = minutes % 5
	return metric
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {

	scaledObjectMetricTargetSizeStr := scaledObject.ScalerMetadata[SCALEDOBJECT_KEY_METRIC_TARGETSIZE]

	targetMetric, err := strconv.Atoi(scaledObjectMetricTargetSizeStr)

	if err != nil {
		targetMetric = METRIC_TARGETSIZE
	}

	metric := getCurrentMetric()
	return &pb.IsActiveResponse{
		Result: metric > targetMetric,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(context.Context, *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {
	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: METRIC_NAME,
			TargetSize: METRIC_TARGETSIZE,
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {

	metric := getCurrentMetric()
	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  METRIC_NAME,
			MetricValue: int64(metric),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {

	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Minute * 1):
			metric := getCurrentMetric()

			if metric > METRIC_TARGETSIZE {
				err := epsServer.Send(&pb.IsActiveResponse{
					Result: true,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func main() {
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":6000")
	pb.RegisterExternalScalerServer(grpcServer, &ExternalScaler{})

	fmt.Println("listenting on :6000")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
