// class.gs —— 类的继承与多态
class Animal {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  speak(): string {
    return `${this.name} makes a sound`;
  }
}

class Dog extends Animal {
  breed: string;

  constructor(name: string, breed: string) {
    super(name);
    this.breed = breed;
  }

  speak(): string {
    return `${this.name} (${this.breed}) barks`;
  }
}

let animals: Animal[] = [
  new Animal("Generic"),
  new Dog("Rex", "Labrador"),
];

for (let a of animals) {
  console.log(a.speak());
}
