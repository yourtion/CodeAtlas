package com.example.myapp.service;

import java.util.List;
import java.util.ArrayList;
import com.example.myapp.model.User;
import com.example.myapp.repository.UserRepository;
import org.springframework.stereotype.Service;

/**
 * User service demonstrating package-aware dependency resolution.
 */
@Service
public class UserService {
    private UserRepository repository;
    
    public UserService(UserRepository repository) {
        this.repository = repository;
    }
    
    public List<User> getAllUsers() {
        return repository.findAll();
    }
    
    public User getUserById(Long id) {
        return repository.findById(id);
    }
}
