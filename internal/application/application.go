package application

type Application interface {
	Run() (err error)

	SetUp() (err error)

	TearDown() (err error)
}