# Test Crystal file for the language server
class Person
  property name : String
  property age : Int32
  
  def initialize(@name : String, @age : Int32)
end
  
  def greet
    puts "Hello, my name is #{@name} and I am #{@age} years old"
end
  
  def birthday
    @age += 1
    puts "Happy birthday! Now I'm #{@age}"
end
  
  def adult?
    @age >= 18
end
end

module Greeting
  def self.welcome(person : Person)
    puts "Welcome, #{person.name}!"
end
end

# Create an instance
john = Person.new("John", 25)
john.greet
john.birthday




# Different ways to print variables in Crystal:

# 1. Simple variable printing
name = "Alice"
age = 30
puts name          # Prints: Alice
puts age           # Prints: 30

# 2. String interpolation (recommended)
puts "Name: #{name}, Age: #{age}"  # Prints: Name: Alice, Age: 30

# 3. Multiple variables in one puts
puts name, age     # Prints each on a new line

# 4. Concatenation with +
puts "Hello " + name + "!"  # Prints: Hello Alice!

# 5. Using string formatting
puts "Age: %d" % age       # Prints: Age: 30

# 6. Pretty printing with p or pp
array = [1, 2, 3, 4, 5]
hash = {"name" => "Bob", "age" => 25}

puts array         # Prints: [1, 2, 3, 4, 5]
p array           # Same as puts but shows Crystal representation
pp hash           # Pretty prints with nice formatting

# 7. Print without newline
print "Loading"
print "."
print "."
print "."
puts " Done!"     # This adds the newline at the end

puts ""

# Test completion - type "john." to see method suggestions
# Test hover - hover over "Person" to see type info
# Test symbols - use Ctrl+Shift+O to see document symbols
