/* Copyright (c) 2021 David Bulkow */

package main

// send email once a week to each user with active loans and Reservations
//            on the morning when a reservation is to expire on that day
//            an hour before a reservation expires
//            on the morning a reservation is to become active on that day
//            an hour before a reservation goes active

type notifier struct{}

func (n *notifier) weekly()            {}
func (n *notifier) daily()             {}
func (n *notifier) send(target string) {}
