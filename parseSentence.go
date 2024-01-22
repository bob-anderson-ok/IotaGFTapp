package main

//if strings.Contains(sentence, "$") {
//	parts := strings.Split(sentence, " ")
//	if len(parts) < 2 {
//		log.Print("split of line on space did not give 2 parts")
//	}
//	sentence := parts[1][:len(parts[1])-1]
//	sc <- sentence
//	switch sentence[0] {
//	case '$':
//		s, nmeaErr := nmea.Decode([]byte(sentence))
//		if nmeaErr != nil {
//			log.Print(nmeaErr)
//		}
//		//fmt.Printf("%#v\n", s)
//		//fmt.Println("Type of s:", reflect.TypeOf(s))
//		switch s.(type) {
//		case nmea.PUBXTime:
//			//fmt.Println("I see the type nmea.PUBXTime")
//		case *nmea.PUBXTime:
//			//fmt.Println(" YEAH I see a &PUBX")
//			//header := s.(*nmea.PUBXTime).Header
//			//timeLoc := s.(*nmea.PUBXTime).TimeOfDay
//			//leapS := s.(*nmea.PUBXTime).Leap_s
//			//fmt.Println("I extracted:", header, timeLoc, leapS)
//		}
//	case 0xB5:
//		s, ubxErr := ubx.Decode([]byte(sentence))
//		if ubxErr != nil {
//			log.Print(ubxErr)
//		}
//		sc <- fmt.Sprintf("Type of s: %v", reflect.TypeOf(s))
//	}
//}
//
//sc <- sentence
