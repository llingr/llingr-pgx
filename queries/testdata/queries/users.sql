-- Example queries for ExampleLoad (example_test.go).

-- name: all-users
SELECT user_id, full_name FROM users;

-- name: user-by-email
SELECT user_id, full_name
FROM   users
WHERE  email = @email;
