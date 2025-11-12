package com.example.myapp.service

import com.example.myapp.model.User
import com.example.myapp.repository.UserRepository

/**
 * User service with business logic.
 * Demonstrates internal project dependencies in Kotlin.
 */
class UserService(private val repository: UserRepository) {
    
    fun getAllUsers(): List<User> {
        return repository.findAll()
    }
    
    fun getUserById(id: Long): User? {
        return repository.findById(id)
    }
    
    fun createUser(user: User) {
        repository.save(user)
    }
    
    fun deleteUser(id: Long) {
        repository.delete(id)
    }
}
