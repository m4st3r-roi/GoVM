package heap

import (
	"GoVM/chapter3-cf/classfile"
)

type MethodRef struct {
	MemberRef
	method *Method
}

func newMethodRef(cp *ConstantPool, methodrefInfo *chapter3_cf.ConstantMethodrefInfo) *MethodRef {
	ref := &MethodRef{}
	ref.cp = cp
	ref.copyMemberRefInfo(&methodrefInfo.ConstantMemberrefInfo)
	return ref
}

func (self *MethodRef) ResolvedMethod() *Method {
	if self.method == nil {
		self.resolveMethodRef()
	}
	return self.method
}

/**
	类 d 通过方法符号引用访问类 c 的某个方法：
	即 d 调用 c.method
		先要解析符号引用得到类 c
		如果 c 是接口，抛异常
		然后根据方法名和描述符查找方法，找不到抛异常
		否则查看 d 是否有权限访问c

	作者笔记：
		捋了一下，c即为方法的主人，d为方法的调用方
		即 class D {
			public void do(){
				c.sayHello();
			}
		}
		形如以上形式

		MethodRef是由classFile常量池中的 ConstantMethodrefInfo 转化而来
		ConstantMethodrefInfo 在编译器，会存储C的class的index
 */
func (self *MethodRef) resolveMethodRef() {

	//这里我第一次写成 : self.class 这里有问题 这里的self.class 是个null 还在排查中
	//这个问题现在的考虑是这样的：由于我们在 7 章节的时候，还没有实现类初始化，
	//我们new只new出了一个FibonacciTest，它调用的system之类的都没有初始化，当然也没法从文件中的常量池解析为运行时常量池
	//
	//调试了一下发现,如果我们写成
	//func newMethodRef(cp *ConstantPool, methodrefInfo *chapter3_cf.ConstantMethodrefInfo) *MethodRef {
	//ref := &MethodRef{}
	//ref.cp = cp
	//ref.class = cp.class   <-  这里是新加的那句
	//ref.copyMemberRefInfo(&methodrefInfo.ConstantMemberrefInfo)
	//return ref
	//}
	//
	//这里的cp 一直都是FibonacciTest，但是它继承中的SymRef的name是来自常量池中MethodInfo的name。
	//而且SymRef的class在没有加载的时候是nil，是需要通过FibonacciTest的classLoader 去加载 outputStream这个类，加载之后 就是下面的c这个类
	//说白了 调用方 d 来自FibonacciTest的常量池，并从常量池中获取的class引用
	// 而 c （outputStream） 需要 这个方法 -> self 的 resolveClass获得。及加载outputStream这个类

	d := self.cp.class

	c := self.ResolvedClass()
	if c.IsInterface() {
		panic("java.lang.IncompatibleClassChangeError")
	}

	method := lookupMethod(c, self.name, self.descriptor)
	if method == nil {
		panic("java.lang.NoSuchMethodError")
	}

	if !method.isAccessibleTo(d) {
		panic("java.lang.IllegalAccessError")
	}

	self.method = method
}

/**
	非接口方法引用的解析
 */
func lookupMethod(class *Class, name, descriptor string) *Method {
	method := LookupMethodInClass(class, name, descriptor)
	if method == nil {
		method = lookupMethodInInterface(class.interfaces, name, descriptor)
	}
	return method
}

/**
	从父类中找方法
 */
func LookupMethodInClass(class *Class, name, descriptor string) *Method {
	for c := class; c != nil; c = c.superClass {
		for _, method := range c.methods {
			if method.name == name && method.descriptor == descriptor {
				return method
			}
		}
	}
	return nil
}

/**
	从接口中找方法
 */
func lookupMethodInInterface(ifaces []*Class, name, descriptor string) *Method {
	for _, iface := range ifaces {
		for _, method := range iface.methods {
			if method.name == name && method.descriptor == descriptor {
				return method
			}
		}
		method := lookupMethodInInterface(iface.interfaces, name, descriptor)
		if method != nil {
			return method
		}
	}
	return nil
}