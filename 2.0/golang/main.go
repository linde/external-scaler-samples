package main

import (
	"context"
	pb "externalscaler-sample/externalscaler"
	"fmt"
	"log"
	"net"
	"strconv"

	"time"

	"google.golang.org/grpc"
)

type ExternalScaler struct{}

const METRIC_NAME = "QUEUE"

const KEY_METRIC_TARGETSIZE = "metricTargetSize"
const DEFAULT_METRIC_TARGETSIZE = 2

const KEY_METRIC_MODULUS = "metricModulus"
const DEFAULT_METRIC_MODULUS = 3

func getCurrentMetric(metricModulus int) (metric int) {
	_, minutes, _ := time.Now().Clock()
	metric = minutes % metricModulus

	log.Printf("getCurrentMetric: %d %% %d: %d", minutes, metricModulus, metric)

	return metric
}

func getScaledObjectKey(scaledObject *pb.ScaledObjectRef, key string, defaultVal int) int {

	if scaledObject == nil {
		return defaultVal
	}
	scaledObjectValStr, ok := scaledObject.ScalerMetadata[key]
	if !ok {
		return defaultVal
	}
	val, err := strconv.Atoi(scaledObjectValStr)
	if err != nil {
		val = defaultVal
	}
	return val
}

func getTargetMetric(scaledObject *pb.ScaledObjectRef) int {
	return getScaledObjectKey(scaledObject, KEY_METRIC_TARGETSIZE,
		DEFAULT_METRIC_TARGETSIZE)
}

func getMetricModulus(scaledObject *pb.ScaledObjectRef) int {
	return getScaledObjectKey(scaledObject,
		KEY_METRIC_MODULUS, DEFAULT_METRIC_MODULUS)
}

func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *pb.ScaledObjectRef) (*pb.IsActiveResponse, error) {

	targetMetric := getTargetMetric(scaledObject)
	metricModulus := getMetricModulus(scaledObject)

	metric := getCurrentMetric(metricModulus)
	log.Printf("IsActive: metric: %d, targetMetric: %d", metric, targetMetric)

	return &pb.IsActiveResponse{
		Result: metric >= targetMetric,
	}, nil
}

func (e *ExternalScaler) GetMetricSpec(_ context.Context, scaledObject *pb.ScaledObjectRef) (*pb.GetMetricSpecResponse, error) {

	targetMetric := getTargetMetric(scaledObject)

	return &pb.GetMetricSpecResponse{
		MetricSpecs: []*pb.MetricSpec{{
			MetricName: METRIC_NAME,
			TargetSize: int64(targetMetric),
		}},
	}, nil
}

func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {

	metricModulus := getMetricModulus(metricRequest.GetScaledObjectRef())
	metric := getCurrentMetric(metricModulus)

	return &pb.GetMetricsResponse{
		MetricValues: []*pb.MetricValue{{
			MetricName:  METRIC_NAME,
			MetricValue: int64(metric),
		}},
	}, nil
}

func (e *ExternalScaler) StreamIsActive(scaledObject *pb.ScaledObjectRef, epsServer pb.ExternalScaler_StreamIsActiveServer) error {

	targetMetric := getTargetMetric(scaledObject)
	metricModulus := getMetricModulus(scaledObject)

	for {
		select {
		case <-epsServer.Context().Done():
			// call cancelled
			return nil
		case <-time.Tick(time.Minute * 1):
			metric := getCurrentMetric(metricModulus)
			log.Printf("StreamIsActive: metric: %d, targetMetric: %d", metric, targetMetric)

			if metric >= targetMetric {
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
