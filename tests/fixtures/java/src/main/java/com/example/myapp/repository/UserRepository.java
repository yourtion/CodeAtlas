package com.example.myapp.repository;

import java.util.List;
import java.util.ArrayList;
import com.example.myapp.model.User;

/**
 * User repository for data access.
 */
public class UserRepository {
    private List<User> users = new ArrayList<>();
    
    public List<User> findAll() {
        return new ArrayList<>(users);
    }
    
    public User findById(Long id) {
        for (User user : users) {
            if (user.getId().equals(id)) {
                return user;
            }
        }
        return null;
    }
    
    public void save(User user) {
        users.add(user);
    }
    
    public void delete(Long id) {
        users.removeIf(user -> user.getId().equals(id));
    }
}
