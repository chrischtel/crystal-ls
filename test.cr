class Calculator
  property value : Float64
  
  def initialize(@value : Float64 = 0.0)
  end
  
  def add(num : Float64)
    @value += num
    self
  end
  
  def multiply(num : Float64)
    @value *= num
    self
  end
  
  def result
    @value
  end
  
  def reset
    @value = 0.0
    self
  end
end

class Person
  property name : String
  property age : Int32
  
  def initialize(@name : String, @age : Int32)
  end
  
  def greet
    puts "Hello, I'm #{@name} and I'm #{@age} years old"
  end
  
  def birthday
    @age += 1
    puts "Happy birthday! Now I'm #{@age}"
  end
  
  def adult?
    @age >= 18
  end
  
  def to_s
    "#{@name} (#{@age})"
  end
end

calc = Calculator.new(10.0)
john = Person.new("John", 25)

class Test
  def self.hello_world : String
    "Hello World"
  end
end

anc = Test.new()


puts "hello"

