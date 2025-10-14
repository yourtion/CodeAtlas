"""
Python test file with various constructs
"""

import json
from typing import List, Optional
from datetime import datetime


class User:
    """
    User model representing a system user
    """
    
    def __init__(self, username: str, email: str):
        """
        Initialize a new User
        
        Args:
            username: The user's username
            email: The user's email address
        """
        self.username = username
        self.email = email
        self.created_at = datetime.now()
    
    def to_dict(self) -> dict:
        """Convert user to dictionary representation"""
        return {
            'username': self.username,
            'email': self.email,
            'created_at': self.created_at.isoformat()
        }
    
    @classmethod
    def from_dict(cls, data: dict) -> 'User':
        """Create User from dictionary"""
        return cls(data['username'], data['email'])


class Repository:
    """
    Repository model for code repositories
    """
    
    def __init__(self, name: str, owner: User):
        self.name = name
        self.owner = owner
        self.files: List[str] = []
    
    def add_file(self, filename: str) -> None:
        """Add a file to the repository"""
        if filename not in self.files:
            self.files.append(filename)
    
    def get_file_count(self) -> int:
        """Get the number of files in repository"""
        return len(self.files)


def process_data(data: dict) -> Optional[str]:
    """
    Process input data and return result
    
    Args:
        data: Input data dictionary
        
    Returns:
        Processed result string or None
    """
    if not data:
        return None
    
    return json.dumps(data)


async def fetch_remote_data(url: str) -> dict:
    """
    Async function to fetch remote data
    
    Args:
        url: The URL to fetch from
        
    Returns:
        Fetched data as dictionary
    """
    # Simulated async operation
    return {'url': url, 'status': 'success'}
