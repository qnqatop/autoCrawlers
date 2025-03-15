package mobilede

// find interesting url https://m.mobile.de/consumer/api/search/reference-data/filters/Car

// data {data: {ab: [{i: "", n: "Beliebig"}, {i: "DRIVER_AIRBAG", n: "Fahrer-Airbag"},…],…}}
//data
//:
//{ab: [{i: "", n: "Beliebig"}, {i: "DRIVER_AIRBAG", n: "Fahrer-Airbag"},…],…}
//ab
//:
//[{i: "", n: "Beliebig"}, {i: "DRIVER_AIRBAG", n: "Fahrer-Airbag"},…]
//ao
//:
//[{i: "PICTURES", n: "Inserate mit Bildern"}, {i: "PRICE_REDUCED", n: "Reduzierter Preis"},…]
//asl
//:
//[{i: "true", n: "zzgl. Lieferungen"}]
//av
//:
//[{i: "INSTANT", n: "Sofort verfügbar"}, {i: "LATER", n: "Mit Lieferzeit"}]
//bat
//:
//[{i: "BATTERY_PURCHASED", n: "inklusive"}, {i: "BATTERY_RENTED", n: "enthalten / Miete"},…]
//bc
//:
//[10, 20, 30, 40, 50, 60, 70, 80, 90, 100, 110, 120, 130, 140, 150]
//bds
//:
//[{i: "EMERGENCY_WHEEL", n: "Notrad"}, {i: "REPAIR_KIT", n: "Pannenkit"},…]
//blt
//:
//[{i: "", n: "Beliebig"}, {i: "ADAPTIVE_BENDING_LIGHTS", n: "Adaptives Kurvenlicht"},…]
//c
//:
//[{i: "OffRoad", n: "SUV / Geländewagen / Pickup"}, {i: "Cabrio", n: "Cabrio / Roadster"},…]
//cc
//:
//[1000, 1200, 1400, 1600, 1800, 2000, 2600, 3000, 5000, 7500, 8000, 9000]
//cht
//:
//[{i: "", n: "Beliebig"}, {i: ":180", n: "Bis zu 3 Stunden"}, {i: ":360", n: "Bis zu 6 Stunden"},…]
//chtf
//:
//[{i: "", n: "Beliebig"}, {i: ":30", n: "Bis zu 30 Minuten"}, {i: ":60", n: "Bis zu 1 Stunde"},…]
//clim
//:
//[{i: "", n: "Beliebig"}, {i: "NO_CLIMATISATION", n: "Keine Klimaanlage oder -automatik"},…]
//cn
//:
//[{i: "", n: "Beliebig"}, {i: "EG", n: "Ägypten"}, {i: "AL", n: "Albanien"}, {i: "AD", n: "Andorra"},…]
//cnc
//:
//[{i: "", n: "Beliebig"}, {i: ":3", n: "3 l/100 km"}, {i: ":5", n: "5 l/100 km"},…]
//con
//:
//[{i: "", n: "Beliebig"}, {i: "NEW", n: "Neu"}, {i: "USED", n: "Gebraucht"}]
//cy
//:
//[3, 4, 6, 8, 10, 12]
//dam
//:
//[{i: "", n: "Alle anzeigen"}, {i: "false", n: "Nicht anzeigen"}, {i: "true", n: "Nur anzeigen"}]
//doc
//:
//[{i: "", n: "Beliebig"}, {i: 1, n: "1 Tag"}, {i: 3, n: "3 Tagen"}, {i: 7, n: "7 Tagen"},…]
//door
//:
//[{i: "", n: "Beliebig"}, {i: "TWO_OR_THREE", n: "2/3"}, {i: "FOUR_OR_FIVE", n: "4/5"},…]
//drl
//:
//[{i: "", n: "Beliebig"}, {i: "LED_RUNNING_LIGHTS", n: "LED-Tagfahrlicht"},…]
//dt
//:
//[{i: "ALL_WHEEL", n: "Allrad"}, {i: "FRONT", n: "Frontantrieb"}, {i: "REAR", n: "Heckantrieb"}]
//ecol
//:
//[{i: "BLACK", n: "Schwarz"}, {i: "BEIGE", n: "Beige"}, {i: "GREY", n: "Grau"},…]
//emc
//:
//[{i: "", n: "Beliebig"}, {i: "EURO1", n: "Euro 1"}, {i: "EURO2", n: "Euro 2"},…]
//ems
//:
//[{i: "", n: "Beliebig"}, {i: "EMISSIONSSTICKER_NONE", n: "1 (Keine)"},…]
//fe
//:
//[{i: "ABS", n: "ABS"}, {i: "AIR_SUSPENSION", n: "Luftfederung"}, {i: "ALARM_SYSTEM", n: "Alarmanlage"},…]
//fe!
//:
//[{i: "EXPORT", n: "Für Gewerbe, Ex-/Import"}]
//fr
//:
//[2025, 2024, 2023, 2022, 2021, 2020, 2019, 2018, 2017, 2016, 2015, 2014, 2013, 2012, 2011, 2010, 2009,…]
//ft
//:
//[{i: "PETROL", n: "Benzin"}, {i: "DIESEL", n: "Diesel"}, {i: "ELECTRICITY", n: "Elektro"},…]
//ftv
//:
//[30, 50, 80, 100, 150]
//gi
//:
//[{i: "", n: "Beliebig"}, {i: "0", n: "Neu"}, {i: "18", n: "18 Monate"}, {i: "12", n: "12 Monate"},…]
//hlt
//:
//[{i: "BI_XENON_HEADLIGHTS", n: "Bi-Xenon Scheinwerfer"}, {i: "LASER_HEADLIGHTS", n: "Laserlicht"},…]
//icol
//:
//[{i: "BEIGE", n: "Beige"}, {i: "BLUE", n: "Blau"}, {i: "BROWN", n: "Braun"}, {i: "GREY", n: "Grau"},…]
//it
//:
//[{i: "ALCANTARA", n: "Alcantara"}, {i: "OTHER_INTERIOR_TYPE", n: "Andere"},…]
//ls
//:
//[{i: ";12:;", n: "Angebote mit Leasing"}]
//lscd
//:
//[{i: "lb", n: "Loyalisierungsprämie"}, {i: "dd", n: "Behindertenrabatt"},…]
//lsdt
//:
//[{i: "bt", n: "Online konfigurierbar"}, {i: "ch", n: "Checkout"}, {i: "cl", n: "Classic"}]
//lsml
//:
//[5000, 10000, 15000, 20000, 25000, 30000, 40000]
//lsrt
//:
//[50, 100, 150, 200, 250, 300, 350, 400, 450, 500, 600, 700, 800, 900, 1000]
//lst
//:
//[{i: "p", n: "Private Nutzung"}, {i: "c", n: "Gewerbliche Nutzung"}]
//lstm
//:
//[12, 18, 24, 30, 36, 42, 48, 54, 60, 72]
//ml
//:
//[5000, 10000, 20000, 30000, 40000, 50000, 60000, 70000, 80000, 90000, 100000, 125000, 150000, 175000,…]
//mnw
//:
//[50, 100, 150, 200]
//mpr
//:
//[{i: "VERY_GOOD_PRICE", n: "Sehr guter Preis"}, {i: "GOOD_PRICE", n: "Guter Preis"},…]
//ms
//:
//[{i: 140, n: "Abarth"}, {i: 203, n: "AC"}, {i: 375, n: "Acura"}, {i: 31930, n: "Aiways"},…]
//nw
//:
//[400, 500, 600, 700, 800, 900, 1000, 1500, 2000, 2500, 3000]
//obs
//:
//[{i: "true", n: "Online-Kauf"}]
//p
//:
//[500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000, 6000, 7000, 8000, 9000, 10000, 11000,…]
//pa
//:
//[{i: "CAM_360_DEGREES", n: "360° Kamera"}, {i: "REAR_VIEW_CAM", n: "Kamera"},…]
//pvo
//:
//[{i: "", n: "Beliebig"}, {i: 1, n: "bis zu 1"}, {i: 2, n: "bis zu 2"}, {i: 3, n: "bis zu 3"},…]
//pw
//:
//[25, 37, 44, 55, 66, 74, 87, 96, 110, 147, 185, 223, 263, 296, 334]
//rad
//:
//[{i: "", n: "Beliebig"}, {i: "DAB_RADIO", n: "Radio DAB"}, {i: "TUNER", n: "Tuner/Radio"}]
//rd
//:
//[10, 20, 50, 100, 200, 500]
//re
//:
//[{i: "", n: "Beliebig"}, {i: "50:", n: "Mind. 50 km"}, {i: "100:", n: "Mind. 100 km"},…]
//rtd
//:
//[{i: "true", n: "Fahrtauglich"}]
//sc
//:
//[2, 3, 4, 5, 6, 7]
//sld
//:
//[{i: "", n: "Beliebig"}, {i: "SLIDING_DOOR_BOTH_SIDED", n: "Schiebetür beidseitig"},…]
//spc
//:
//[{i: "", n: "Beliebig"}, {i: "ADAPTIVE_CRUISE_CONTROL", n: "Abstandstempomat"},…]
//sr
//:
//[{i: "", n: "Beliebig"}, {i: "3:", n: "ab 3 Sterne"}, {i: "4:", n: "ab 4 Sterne"},…]
//st
//:
//[{i: "", n: "Beliebig"}, {i: "DEALER", n: "Händler"}, {i: "FSBO", n: "Privatanbieter"},…]
//subc
//:
//[{i: "EMPLOYEES_CAR", n: "Jahreswagen"}, {i: "CLASSIC", n: "Oldtimer"},…]
//tct
//:
//[{i: "", n: "Beliebig"}, {i: "TRAILER_COUPLING_FIX", n: "Fest, abnehmbar oder schwenkbar"},…]
//tlb
//:
//[500, 1000, 1500, 2000, 2500, 3000, 3500]
//tlu
//:
//[300, 350, 400, 450, 500, 550, 600, 650, 700, 750, 800]
//tr
//:
//[{i: "AUTOMATIC_GEAR", n: "Automatik"}, {i: "SEMIAUTOMATIC_GEAR", n: "Halbautomatik"},…]
//ucs
//:
//[{i: "", n: "Bitte wählen"}, {i: "FCA", n: "Alfa Romeo Certified"},…]
//ucsa
//:
//[{i: "true", n: "Inserate mit Qualitätssiegel"}]
//vat
//:
//[{i: "", n: "Beliebig"}, {i: 1, n: "MwSt. ausweisbar"}, {i: 2, n: "MwSt. nicht ausweisbar"}]
