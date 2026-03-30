import os
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor


def init_jaeger_tracing(service_name):
    """Initialize Jaeger tracing for the Flask application"""
    jaeger_endpoint = os.getenv("JAEGER_ENDPOINT", "http://localhost:4318")
    
    # Create OTLP exporter
    otlp_exporter = OTLPSpanExporter(endpoint=f"{jaeger_endpoint}/v1/traces")
    
    # Create trace provider
    trace_provider = TracerProvider(
        resource=Resource.create({
            SERVICE_NAME: service_name,
        })
    )
    
    # Add span processor
    trace_provider.add_span_processor(BatchSpanProcessor(otlp_exporter))
    
    # Set global trace provider
    trace.set_tracer_provider(trace_provider)
    
    return trace_provider


def instrument_app(app, service_name):
    """Instrument a Flask app with tracing"""
    # Initialize Jaeger
    tp = init_jaeger_tracing(service_name)
    
    # Instrument Flask
    FlaskInstrumentor().instrument_app(app)
    
    # Instrument requests library
    RequestsInstrumentor().instrument()
    
    return tp
