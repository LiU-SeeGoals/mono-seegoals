/*
 * spiSpeed.c:
 *	Code to measure the SPI speed/latency.
 *	Copyright (c) 2014 Gordon Henderson
 ***********************************************************************
 * This file is part of wiringPi:
 *	https://github.com/WiringPi/WiringPi/
 *
 *    wiringPi is free software: you can redistribute it and/or modify
 *    it under the terms of the GNU Lesser General Public License as
 *    published by the Free Software Foundation, either version 3 of the
 *    License, or (at your option) any later version.
 *
 *    wiringPi is distributed in the hope that it will be useful,
 *    but WITHOUT ANY WARRANTY; without even the implied warranty of
 *    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *    GNU Lesser General Public License for more details.
 *
 *    You should have received a copy of the GNU Lesser General Public
 *    License along with wiringPi.
 *    If not, see <http://www.gnu.org/licenses/>.
 ***********************************************************************
 */


#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
//#include <fcntl.h>
//#include <sys/ioctl.h>
//#include <linux/spi/spidev.h>

#include <wiringPi.h>
#include <wiringPiSPI.h>

#define	TRUE	(1==1)
#define	FALSE	(!TRUE)

#define	SPI_CHAN		0
#define	NUM_TIMES		100
#define	MAX_SIZE		(1024)

static int myFd ;


void spiSetup (int speed)
{
  if ((myFd = wiringPiSPISetupMode (SPI_CHAN, speed, 0)) < 0)
  {
    fprintf (stderr, "Can't open the SPI bus: %s\n", strerror (errno)) ;
    exit (EXIT_FAILURE) ;
  }
}

void printBits(unsigned char x){
	unsigned char num = 8;

	for(int i = 1; i <= num; i++)
	    printf("%d", (x >> (num - i)) & 1);
    printf(" ");
}

int main (void)
{
  int spiFail ;
  unsigned char myData[65];

  int speed = 2;
  int size = 65;
  wiringPiSetup () ;

  spiSetup (speed * 1000000) ;
  //spiSetup (10000) ;
  while(1){
	  spiFail = FALSE ;
	  if (wiringPiSPIDataRW (SPI_CHAN, myData, size) == -1)
	  {
		printf ("SPI failure: %s\n", strerror (errno)) ;
		spiFail = TRUE ;
	  }
	  unsigned char len = myData[0];
	  for (int i = 0; i < len + 1; i ++)
	  {
		printBits(myData[i]);
	  }
	  for (int i = len + 1; i < size; i ++)
	  {
		  if (myData[i] != 0)
		  {
			  printf("shit\n");
		  }
	  }
	  printf("\n");
  }
  close (myFd) ;

  return 0 ;
}
