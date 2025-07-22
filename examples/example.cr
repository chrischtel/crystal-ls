# Example Crystal program to test the language server

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
  
  def self.farewell(name : String)
    puts "Goodbye, #{name}!"
  end
end

# Create instances and test methods
john = Person.new("John", 25)
jane = Person.new("Jane", 17)

john.greet
jane.greet

puts "John is adult: #{john.adult?}"
puts "Jane is adult: #{jane.adult?}"

jane.birthday
puts "Jane is adult now: #{jane.adult?}"

Greeting.welcome(john)
Greeting.farewell("Everyone")

# Array operations
numbers = [1, 2, 3, 4, 5]
squares = numbers.map { |n| n ** 2 }
puts "Squares: #{squares}"

# String operations
text = "Hello, Crystal!"
puts text.upcase
puts text.size
puts text.includes?("Crystal")

# Hash operations
scores = {"Alice" => 95, "Bob" => 87, "Charlie" => 92}
scores.each do |name, score|
  puts "#{name}: #{score}"
end

# Conditional logic
weather = "sunny"
case weather
when "sunny"
  puts "Great day for a walk!"
when "rainy"
  puts "Better stay inside"
else
  puts "Not sure about the weather"
end
