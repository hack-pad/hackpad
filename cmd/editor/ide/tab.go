package ide

type Tabber interface {
	Titles() <-chan string
}
