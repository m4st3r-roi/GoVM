package heap

import (
	"GoVM/chapter3-cf/classfile"
	"strings"
)

/**
	我们需要注意一下JVM中的class
	我们这里的class，是运行时常量池里的class

	我们的java.lang.Class 继承 java.lang.Object
	java.lang.Object.class 又需要有一个 java.lang.Class（java.lang.Object.class 是存放在堆内存中的，相当于java.lang.Class的实例）
	那么问题来了，是现有的java.lang.Object还是先有的java.lang.Class

	这种问题，非常类似于spring解决set方式的循环依赖，都是通过在“混沌态”的时候解决问题。
	JVM也是这样的，在这两个class在运行时常量池中还没完全初始化的时候，暴露出引用，来解决这个问题，这样我们就有了java世界里最最基础的两个class
**/
type Class struct {
	//类访问标识符
	accessFlags uint16
	//thisClassName
	name           string
	superClassName string
	interfaceNames []string
	constantPool   *ConstantPool
	fields         []*Field
	methods        []*Method
	loader         *ClassLoader
	superClass     *Class
	interfaces     []*Class
	//实例变量(及private String name)占据的空间大小
	InstanceSlotCount uint
	//类变量(及static类型的变量)占据的空间大小
	staticSlotCount uint
	staticVars      Slots
	//类的 <clinit> 方法是否已经开始执行
	initStarted bool
	//与一个java中的java.lang.Class对应，而这个struct本身指的是虚拟机中的方法区中class的相关数据
	jClass     *Object
	sourceFile string
}

func newClass(cf *chapter3_cf.ClassFile) *Class {
	class := &Class{}
	class.accessFlags = cf.AccessFlags()
	class.name = cf.ClassName()
	class.superClassName = cf.SuperClassName()
	class.interfaceNames = cf.InterfaceNames()
	class.constantPool = newConstantPool(class, cf.ConstantPool())
	class.fields = newFields(class, cf.Fields())
	class.methods = newMethods(class, cf.Methods())
	class.sourceFile = getSourceFile(cf)
	return class
}

func getSourceFile(cf *chapter3_cf.ClassFile) string {
	if sfAttr := cf.SourceFileAttribute(); sfAttr != nil {
		return sfAttr.FileName()
	}
	return "Unknown"
}

func (self *Class) IsPublic() bool {
	return 0 != self.accessFlags&ACC_PUBLIC
}

func (self *Class) IsFinal() bool {
	return 0 != self.accessFlags&ACC_FINAL
}
func (self *Class) IsSuper() bool {
	return 0 != self.accessFlags&ACC_SUPER
}
func (self *Class) IsInterface() bool {
	return 0 != self.accessFlags&ACC_INTERFACE
}
func (self *Class) IsAbstract() bool {
	return 0 != self.accessFlags&ACC_ABSTRACT
}
func (self *Class) IsSynthetic() bool {
	return 0 != self.accessFlags&ACC_SYNTHETIC
}
func (self *Class) IsAnnotation() bool {
	return 0 != self.accessFlags&ACC_ANNOTATION
}
func (self *Class) IsEnum() bool {
	return 0 != self.accessFlags&ACC_ENUM
}

func (self *Class) StartInit() {
	self.initStarted = true
}

// getters start
func (self *Class) ConstantPool() *ConstantPool {
	return self.constantPool
}
func (self *Class) StaticVars() Slots {
	return self.staticVars
}

func (self *Class) Name() string {
	return self.name
}

func (self *Class) Fields() []*Field {
	return self.fields
}
func (self *Class) Methods() []*Method {
	return self.methods
}

func (self *Class) SuperClass() *Class {
	return self.superClass
}

func (self *Class) InitStarted() bool {
	return self.initStarted
}

func (self *Class) Loader() *ClassLoader {
	return self.loader
}

func (self *Class) JClass() *Object {
	return self.jClass
}

// getters end

func (self *Class) JavaName() string {
	return strings.Replace(self.name, "/", ".", -1)
}

func (self *Class) NewObject() *Object {
	return newObject(self)
}

func (self *Class) ArrayClass() *Class {
	arrayClassName := getArrayClassName(self.name)
	return self.loader.LoadClass(arrayClassName)
}

/**
是否有权限访问
*/
func (self *Class) isAccessibleTo(other *Class) bool {
	return self.IsPublic() || self.GetPackageName() == other.GetPackageName()
}

// other extends self
func (self *Class) IsSuperClassOf(other *Class) bool {
	return other.IsSubClassOf(self)
}

/**
是否是基本类型
*/
func (self *Class) IsPrimitive() bool {
	_, ok := primitiveTypes[self.name]
	return ok
}

// self extends other
func (self *Class) IsSubClassOf(other *Class) bool {
	for c := self.superClass; c != nil; c = c.superClass {
		if c == other {
			return true
		}
	}
	return false
}

func (self *Class) isJlObject() bool {
	return self.name == "java/lang/Object"
}
func (self *Class) isJlCloneable() bool {
	return self.name == "java/lang/Cloneable"
}
func (self *Class) isJioSerializable() bool {
	return self.name == "java/io/Serializable"
}

func (self *Class) GetPackageName() string {
	if i := strings.LastIndex(self.name, "/"); i >= 0 {
		return self.name[:i]
	}
	return ""
}

func (self *Class) getMethod(name, descriptor string, isStatic bool) *Method {
	for c := self; c != nil; c = c.superClass {
		for _, method := range c.methods {
			if method.IsStatic() == isStatic &&
				method.name == name &&
				method.descriptor == descriptor {

				return method
			}
		}
	}
	return nil
}

func (self *Class) GetMainMethod() *Method {
	return self.getStaticMethod("main", "([Ljava/lang/String;)V")
}

func (self *Class) GetInstanceMethod(name, descriptor string) *Method {
	return self.getMethod(name, descriptor, false)
}

func (self *Class) GetRefVar(fieldName, fieldDescriptor string) *Object {
	field := self.getField(fieldName, fieldDescriptor, true)
	return self.staticVars.GetRef(field.slotId)
}
func (self *Class) SetRefVar(fieldName, fieldDescriptor string, ref *Object) {
	field := self.getField(fieldName, fieldDescriptor, true)
	self.staticVars.SetRef(field.slotId, ref)
}

/**
根据字段名、描述符以及是否是static来查找方法
*/
func (self *Class) getField(name, descriptor string, isStatic bool) *Field {
	for c := self; c != nil; c = c.superClass {
		for _, field := range c.fields {
			if field.IsStatic() == isStatic && field.name == name && field.descriptor == descriptor {
				return field
			}
		}
	}
	return nil
}

func (self *Class) getStaticMethod(name, descriptor string) *Method {
	for _, method := range self.methods {
		if method.IsStatic() && method.name == name && method.descriptor == descriptor {
			return method
		}
	}
	return nil
}

func (self *Class) GetClinitMethod() *Method {
	return self.getStaticMethod("<clinit>", "()V")
}

func (self *Class) SourceFile() string {
	return self.sourceFile
}
