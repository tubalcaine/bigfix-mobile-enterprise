#! /usr/bin/perl

use strict;

while (<>) {
	chomp;
	my $p1 = $_;
	my $p2 = $_;
	$p2 =~ s/TBD/bes/;
	print "mv '$p1' '$p2'\n";
}
	
