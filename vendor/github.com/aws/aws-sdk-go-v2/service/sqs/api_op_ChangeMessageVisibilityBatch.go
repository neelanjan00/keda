// Code generated by smithy-go-codegen DO NOT EDIT.

package sqs

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Changes the visibility timeout of multiple messages. This is a batch version of
// ChangeMessageVisibility . The result of the action on each message is reported
// individually in the response. You can send up to 10 ChangeMessageVisibility
// requests with each ChangeMessageVisibilityBatch action. Because the batch
// request can result in a combination of successful and unsuccessful actions, you
// should check for batch errors even when the call returns an HTTP status code of
// 200 .
func (c *Client) ChangeMessageVisibilityBatch(ctx context.Context, params *ChangeMessageVisibilityBatchInput, optFns ...func(*Options)) (*ChangeMessageVisibilityBatchOutput, error) {
	if params == nil {
		params = &ChangeMessageVisibilityBatchInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "ChangeMessageVisibilityBatch", params, optFns, c.addOperationChangeMessageVisibilityBatchMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*ChangeMessageVisibilityBatchOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type ChangeMessageVisibilityBatchInput struct {

	// Lists the receipt handles of the messages for which the visibility timeout must
	// be changed.
	//
	// This member is required.
	Entries []types.ChangeMessageVisibilityBatchRequestEntry

	// The URL of the Amazon SQS queue whose messages' visibility is changed. Queue
	// URLs and names are case-sensitive.
	//
	// This member is required.
	QueueUrl *string

	noSmithyDocumentSerde
}

// For each message in the batch, the response contains a
// ChangeMessageVisibilityBatchResultEntry tag if the message succeeds or a
// BatchResultErrorEntry tag if the message fails.
type ChangeMessageVisibilityBatchOutput struct {

	// A list of BatchResultErrorEntry items.
	//
	// This member is required.
	Failed []types.BatchResultErrorEntry

	// A list of ChangeMessageVisibilityBatchResultEntry items.
	//
	// This member is required.
	Successful []types.ChangeMessageVisibilityBatchResultEntry

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationChangeMessageVisibilityBatchMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson10_serializeOpChangeMessageVisibilityBatch{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson10_deserializeOpChangeMessageVisibilityBatch{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "ChangeMessageVisibilityBatch"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addOpChangeMessageVisibilityBatchValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opChangeMessageVisibilityBatch(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opChangeMessageVisibilityBatch(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "ChangeMessageVisibilityBatch",
	}
}
