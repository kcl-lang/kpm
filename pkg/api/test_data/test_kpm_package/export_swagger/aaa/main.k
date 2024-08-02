import bbb

schema A:
    b: bbb.B

a = A {
    b: bbb.B {
        name = "name"
    }
}
