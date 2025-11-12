package com.example.myapp.service;

import java.util.List;
import com.example.myapp.model.User;
import com.example.myapp.repository.UserRepository;

/**
 * User service with business logic.
 * Demonstrates internal project dependencies.
 */
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
    
    public void createUser(User user) {
        repository.save(user);
    }
    
    public void deleteUser(Long id) {
        repository.delete(id);
    }
}
