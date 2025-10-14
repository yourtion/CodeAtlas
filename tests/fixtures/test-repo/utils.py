"""
Utility functions for the test application
"""

from typing import List, Callable
from functools import wraps


def retry(max_attempts: int = 3):
    """
    Decorator for retrying failed operations
    
    Args:
        max_attempts: Maximum number of retry attempts
    """
    def decorator(func: Callable) -> Callable:
        @wraps(func)
        def wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise
                    continue
        return wrapper
    return decorator


@retry(max_attempts=5)
def unreliable_operation(value: int) -> int:
    """
    An operation that might fail
    
    Args:
        value: Input value
        
    Returns:
        Processed value
    """
    return value * 2


class DataProcessor:
    """Process data with various transformations"""
    
    def __init__(self, config: dict):
        self.config = config
    
    def transform(self, data: List[str]) -> List[str]:
        """Transform data based on configuration"""
        return [item.upper() for item in data]
    
    def filter_items(self, data: List[str], predicate: Callable) -> List[str]:
        """Filter items using predicate function"""
        return [item for item in data if predicate(item)]


# Module-level constants
MAX_SIZE = 1024
DEFAULT_TIMEOUT = 30
