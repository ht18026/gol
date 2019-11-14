package main

import (
	"fmt"
	"strconv"
	"strings"
)



// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}


	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {
		//make a 2-D array to store the information for next round
		nextWorld := make([][]byte, p.imageHeight)
		for i := range nextWorld {
			nextWorld[i] = make([]byte, p.imageWidth)
		}
		//create a for loop go through all cells in the world
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				//create a int value that counts how many alive neighbours does a cell have
				aliveNeighbours := 0
				//go through the 3x3 matrix which centred at world[y][x]
				//this is equivalent to go through all neighbours of a cell and the cell itself
				for i := -1; i < 2; i++{
					for j := -1; j < 2; j++{
						if i == 0 && j == 0{continue}                          //I don't care if the cell is alive or dead at this stage

						if world[(y + i + p.imageHeight) % p.imageHeight][(x + j + p.imageWidth) % p.imageWidth] == 255{                  //if there is an alive neighbour, the count of alive neighbours increase by 1
							aliveNeighbours += 1
						}
					}
				}
				if world[y][x] == 255{
					if aliveNeighbours < 2 || aliveNeighbours > 3{                  //if the cell itself is alive, check the neighbours:
						nextWorld[y][x] = 0                                         //if it has <2 or>3 alive neighbours, it will die in nextWorld :(
					} else{nextWorld[y][x] = 255}                                   //if it has =2 or =3 alive neighbours, it will survive in nextWorld :)
				}
				if world[y][x] == 0{
					if aliveNeighbours == 3{                                        //if the cell itself is dead, check the neighbours:
						nextWorld[y][x] = 255                                       //if it has =3 neighbours, it will become alive in nextWorld ;)
					}else{nextWorld[y][x] = 0}
				}
			}
		}
		for y:=0; y< p.imageHeight;y++{                                            //copy all the shits in nextWorld into world
			for x:=0; x<p.imageWidth;x++{                                          //so that world get into next turn
				world[y][x] = nextWorld[y][x]
			}
		}
	}

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}