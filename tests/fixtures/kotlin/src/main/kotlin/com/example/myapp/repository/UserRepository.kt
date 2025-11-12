package com.example.myapp.repository

import com.example.myapp.model.User

/**
 * User repository for data access.
 */
class UserRepository {
    private val users = mutableListOf<User>()
    
    fun findAll(): List<User> {
        return users.toList()
    }
    
    fun findById(id: Long): User? {
        return users.find { it.id == id }
    }
    
    fun save(user: User) {
        users.add(user)
    }
    
    fun delete(id: Long) {
        users.removeIf { it.id == id }
    }
}
