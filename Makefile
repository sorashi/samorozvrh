all: build-all

build-all: build-solver build-server build-frontend

build-solver:
	cd solver && sudo make -j

build-server:
	cd server && make -j

build-frontend:
	cd frontend && make -j
