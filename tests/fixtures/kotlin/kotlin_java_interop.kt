package com.example.myapp.service

import java.util.ArrayList
import java.util.List
import com.example.myapp.model.User
import com.example.myapp.repository.UserRepository
import org.springframework.stereotype.Service
import kotlin.collections.List as KList

/**
 * User service demonstrating Kotlin-Java interop.
 */
@Service
class UserService(private val repository: UserRepository) {
    
    /**
     * Gets all users using Java List.
     */
    fun getAllUsers(): List<User> {
        return repository.findAll()
    }
    
    /**
     * Gets all users using Kotlin List.
     */
    fun getAllUsersKotlin(): KList<User> {
        return repository.findAll().toList()
    }
    
    /**
     * Gets user by ID.
     */
    fun getUserById(id: Long): User? {
        return repository.findById(id)
    }
}
