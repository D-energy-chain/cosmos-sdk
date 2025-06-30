package cli //nolint:misspell

import (
	"io"
	"strings"
	"unicode"
)

// CountryCode - country code (254 countries). Three codes present, for example Russia == RU == RUS == 643.
type CountryCode int64 // int64 for database/sql/driver.Valuer compatibility

// TypeCountryCode for Typer interface
const TypeCountryCode string = "countries.CountryCode"

// TypeCountry for Typer interface
const TypeCountry string = "countries.Country"

// UnknownMsg - unknown return message
const UnknownMsg string = "Unknown"

// Digit ISO 3166-1. Three codes present, for example Russia == RU == RUS == 643.
const (
	// Unknown                                CountryCode = 0
	Unknown CountryCode = 0
	// International                          CountryCode = 999
	International CountryCode = 999
	// Albania                                CountryCode = 8
	Albania CountryCode = 8
	// Algeria                                CountryCode = 12
	Algeria CountryCode = 12
	// AmericanSamoa                          CountryCode = 16
	AmericanSamoa CountryCode = 16
	// Andorra                                CountryCode = 20
	Andorra CountryCode = 20
	// Angola                                 CountryCode = 24
	Angola CountryCode = 24
	// Anguilla                               CountryCode = 660
	Anguilla CountryCode = 660
	// Antarctica                             CountryCode = 10
	Antarctica CountryCode = 10
	// AntiguaAndBarbuda                      CountryCode = 28
	AntiguaAndBarbuda CountryCode = 28
	// Argentina                              CountryCode = 32
	Argentina CountryCode = 32
	// Armenia                                CountryCode = 51
	Armenia CountryCode = 51
	// Aruba                                  CountryCode = 533
	Aruba CountryCode = 533
	// Australia                              CountryCode = 36
	Australia CountryCode = 36
	// Austria                                CountryCode = 40
	Austria CountryCode = 40
	// Azerbaijan                             CountryCode = 31
	Azerbaijan CountryCode = 31
	// Bahamas                                CountryCode = 44
	Bahamas CountryCode = 44
	// Bahrain                                CountryCode = 48
	Bahrain CountryCode = 48
	// Bangladesh                             CountryCode = 50
	Bangladesh CountryCode = 50
	// Barbados                               CountryCode = 52
	Barbados CountryCode = 52
	// Belarus                                CountryCode = 112
	Belarus CountryCode = 112
	// Belgium                                CountryCode = 56
	Belgium CountryCode = 56
	// Belize                                 CountryCode = 84
	Belize CountryCode = 84
	// Benin                                  CountryCode = 204
	Benin CountryCode = 204
	// Bermuda                                CountryCode = 60
	Bermuda CountryCode = 60
	// Bhutan                                 CountryCode = 64
	Bhutan CountryCode = 64
	// Bolivia                                CountryCode = 68
	Bolivia CountryCode = 68
	// BosniaAndHerzegovina                   CountryCode = 70
	BosniaAndHerzegovina CountryCode = 70
	// Botswana                               CountryCode = 72
	Botswana CountryCode = 72
	// Bouvet                                 CountryCode = 74
	Bouvet CountryCode = 74
	// Brazil                                 CountryCode = 76
	Brazil CountryCode = 76
	// BritishIndianOceanTerritory            CountryCode = 86
	BritishIndianOceanTerritory CountryCode = 86
	// Brunei                                 CountryCode = 96
	Brunei CountryCode = 96
	// Bulgaria                               CountryCode = 100
	Bulgaria CountryCode = 100
	// BurkinaFaso                            CountryCode = 854
	BurkinaFaso CountryCode = 854
	// Burundi                                CountryCode = 108
	Burundi CountryCode = 108
	// Cambodia                               CountryCode = 116
	Cambodia CountryCode = 116
	// Cameroon                               CountryCode = 120
	Cameroon CountryCode = 120
	// Canada                                 CountryCode = 124
	Canada CountryCode = 124
	// CapeVerde                              CountryCode = 132
	CapeVerde CountryCode = 132
	// CaboVerde                              CountryCode = 132
	CaboVerde CountryCode = 132
	// CaymanIslands                          CountryCode = 136
	CaymanIslands CountryCode = 136
	// CentralAfricanRepublic                 CountryCode = 140
	CentralAfricanRepublic CountryCode = 140
	// Chad                                   CountryCode = 148
	Chad CountryCode = 148
	// ChannelIslands                         CountryCode = 830
	ChannelIslands CountryCode = 830
	// Chile                                  CountryCode = 152
	Chile CountryCode = 152
	// China                                  CountryCode = 156
	China CountryCode = 156
	// ChristmasIsland                        CountryCode = 162
	ChristmasIsland CountryCode = 162
	// CocosIslands                           CountryCode = 166
	CocosIslands CountryCode = 166
	// Colombia                               CountryCode = 170
	Colombia CountryCode = 170
	// Comoros                                CountryCode = 174
	Comoros CountryCode = 174
	// Congo                                  CountryCode = 178
	Congo CountryCode = 178
	// CongoDemocraticRepublic                CountryCode = 180
	CongoDemocraticRepublic CountryCode = 180
	// CongoDemocracticRepublic - deprecated
	CongoDemocracticRepublic CountryCode = CongoDemocraticRepublic
	// CookIslands                            CountryCode = 184
	CookIslands CountryCode = 184
	// CostaRica                              CountryCode = 188
	CostaRica CountryCode = 188
	// CoteDIvoire                            CountryCode = 384
	CoteDIvoire CountryCode = 384
	// IvoryCoast                             CountryCode = 384
	IvoryCoast CountryCode = 384
	// Croatia                                CountryCode = 191
	Croatia CountryCode = 191
	// Cuba                                   CountryCode = 192
	Cuba CountryCode = 192
	// Cyprus                                 CountryCode = 196
	Cyprus CountryCode = 196
	// CzechRepublic                          CountryCode = 203
	CzechRepublic CountryCode = 203
	// Denmark                                CountryCode = 208
	Denmark CountryCode = 208
	// Djibouti                               CountryCode = 262
	Djibouti CountryCode = 262
	// Dominica                               CountryCode = 212
	Dominica CountryCode = 212
	// DominicanRepublic                      CountryCode = 214
	DominicanRepublic CountryCode = 214
	// Ecuador                                CountryCode = 218
	Ecuador CountryCode = 218
	// Egypt                                  CountryCode = 818
	Egypt CountryCode = 818
	// ElSalvador                             CountryCode = 222
	ElSalvador CountryCode = 222
	// EquatorialGuinea                       CountryCode = 226
	EquatorialGuinea CountryCode = 226
	// Eritrea                                CountryCode = 232
	Eritrea CountryCode = 232
	// Estonia                                CountryCode = 233
	Estonia CountryCode = 233
	// Ethiopia                               CountryCode = 231
	Ethiopia CountryCode = 231
	// FaroeIslands                           CountryCode = 234
	FaroeIslands CountryCode = 234
	// FalklandIslands                        CountryCode = 238
	FalklandIslands CountryCode = 238
	// Fiji                                   CountryCode = 242
	Fiji CountryCode = 242
	// Finland                                CountryCode = 246
	Finland CountryCode = 246
	// France                                 CountryCode = 250
	France CountryCode = 250
	// FrenchGuiana                           CountryCode = 254
	FrenchGuiana CountryCode = 254
	// FrenchPolynesia                        CountryCode = 258
	FrenchPolynesia CountryCode = 258
	// FrenchSouthernTerritories              CountryCode = 260
	FrenchSouthernTerritories CountryCode = 260
	// Gabon                                  CountryCode = 266
	Gabon CountryCode = 266
	// Gambia                                 CountryCode = 270
	Gambia CountryCode = 270
	// Georgia                                CountryCode = 268
	Georgia CountryCode = 268
	// Germany                                CountryCode = 276
	Germany CountryCode = 276
	// Ghana                                  CountryCode = 288
	Ghana CountryCode = 288
	// Gibraltar                              CountryCode = 292
	Gibraltar CountryCode = 292
	// Greece                                 CountryCode = 300
	Greece CountryCode = 300
	// Greenland                              CountryCode = 304
	Greenland CountryCode = 304
	// Grenada                                CountryCode = 308
	Grenada CountryCode = 308
	// Guadeloupe                             CountryCode = 312
	Guadeloupe CountryCode = 312
	// Guam                                   CountryCode = 316
	Guam CountryCode = 316
	// Guatemala                              CountryCode = 320
	Guatemala CountryCode = 320
	// Guinea                                 CountryCode = 324
	Guinea CountryCode = 324
	// GuineaBissau                           CountryCode = 624
	GuineaBissau CountryCode = 624
	// Guyana                                 CountryCode = 328
	Guyana CountryCode = 328
	// Haiti                                  CountryCode = 332
	Haiti CountryCode = 332
	// HeardIslandAndMcDonaldIslands          CountryCode = 334
	HeardIslandAndMcDonaldIslands CountryCode = 334
	// HeardIslandandMcDonaldIslands - deprecated
	HeardIslandandMcDonaldIslands CountryCode = HeardIslandAndMcDonaldIslands
	// Honduras                               CountryCode = 340
	Honduras CountryCode = 340
	// HongKong                               CountryCode = 344
	HongKong CountryCode = 344
	// Hungary                                CountryCode = 348
	Hungary CountryCode = 348
	// Iceland                                CountryCode = 352
	Iceland CountryCode = 352
	// India                                  CountryCode = 356
	India CountryCode = 356
	// Indonesia                              CountryCode = 360
	Indonesia CountryCode = 360
	// Iran                                   CountryCode = 364
	Iran CountryCode = 364
	// Iraq                                   CountryCode = 368
	Iraq CountryCode = 368
	// Ireland                                CountryCode = 372
	Ireland CountryCode = 372
	// IsleOfMan                              CountryCode = 833
	IsleOfMan CountryCode = 833
	// Israel                                 CountryCode = 376
	Israel CountryCode = 376
	// Italy                                  CountryCode = 380
	Italy CountryCode = 380
	// Jamaica                                CountryCode = 388
	Jamaica CountryCode = 388
	// Japan                                  CountryCode = 392
	Japan CountryCode = 392
	// Jordan                                 CountryCode = 400
	Jordan CountryCode = 400
	// Kazakhstan                             CountryCode = 398
	Kazakhstan CountryCode = 398
	// Kenya                                  CountryCode = 404
	Kenya CountryCode = 404
	// Kiribati                               CountryCode = 296
	Kiribati CountryCode = 296
	// Korea                                  CountryCode = 410
	Korea CountryCode = 410
	// KoreaNorth                             CountryCode = 408
	KoreaNorth CountryCode = 408
	// Kuwait                                 CountryCode = 414
	Kuwait CountryCode = 414
	// Kyrgyzstan                             CountryCode = 417
	Kyrgyzstan CountryCode = 417
	// Laos                                   CountryCode = 418
	Laos CountryCode = 418
	// Latvia                                 CountryCode = 428
	Latvia CountryCode = 428
	// Lebanon                                CountryCode = 422
	Lebanon CountryCode = 422
	// Lesotho                                CountryCode = 426
	Lesotho CountryCode = 426
	// Liberia                                CountryCode = 430
	Liberia CountryCode = 430
	// Libya                                  CountryCode = 434
	Libya CountryCode = 434
	// Liechtenstein                          CountryCode = 438
	Liechtenstein CountryCode = 438
	// Lithuania                              CountryCode = 440
	Lithuania CountryCode = 440
	// Luxembourg                             CountryCode = 442
	Luxembourg CountryCode = 442
	// Macau                                  CountryCode = 446
	Macau CountryCode = 446
	// Macao                                  CountryCode = 446
	Macao CountryCode = 446
	// Macedonia                              CountryCode = 807
	Macedonia CountryCode = 807
	// Madagascar                             CountryCode = 450
	Madagascar CountryCode = 450
	// Malawi                                 CountryCode = 454
	Malawi CountryCode = 454
	// Malaysia                               CountryCode = 458
	Malaysia CountryCode = 458
	// Maldives                               CountryCode = 462
	Maldives CountryCode = 462
	// Mali                                   CountryCode = 466
	Mali CountryCode = 466
	// Malta                                  CountryCode = 470
	Malta CountryCode = 470
	// MarshallIslands                        CountryCode = 584
	MarshallIslands CountryCode = 584
	// Martinique                             CountryCode = 474
	Martinique CountryCode = 474
	// Mauritania                             CountryCode = 478
	Mauritania CountryCode = 478
	// Mauritius                              CountryCode = 480
	Mauritius CountryCode = 480
	// Mayotte                                CountryCode = 175
	Mayotte CountryCode = 175
	// Mexico                                 CountryCode = 484
	Mexico CountryCode = 484
	// Micronesia                             CountryCode = 583
	Micronesia CountryCode = 583
	// Moldova                                CountryCode = 498
	Moldova CountryCode = 498
	// Monaco                                 CountryCode = 492
	Monaco CountryCode = 492
	// Mongolia                               CountryCode = 496
	Mongolia CountryCode = 496
	// Montserrat                             CountryCode = 500
	Montserrat CountryCode = 500
	// Morocco                                CountryCode = 504
	Morocco CountryCode = 504
	// Mozambique                             CountryCode = 508
	Mozambique CountryCode = 508
	// Myanmar                                CountryCode = 104
	Myanmar CountryCode = 104
	// Namibia                                CountryCode = 516
	Namibia CountryCode = 516
	// Nauru                                  CountryCode = 520
	Nauru CountryCode = 520
	// Nepal                                  CountryCode = 524
	Nepal CountryCode = 524
	// Netherlands                            CountryCode = 528
	Netherlands CountryCode = 528
	// NetherlandsAntilles                    CountryCode = 530
	NetherlandsAntilles CountryCode = 530
	// NewCaledonia                           CountryCode = 540
	NewCaledonia CountryCode = 540
	// NewZealand                             CountryCode = 554
	NewZealand CountryCode = 554
	// Nicaragua                              CountryCode = 558
	Nicaragua CountryCode = 558
	// Niger                                  CountryCode = 562
	Niger CountryCode = 562
	// Nigeria                                CountryCode = 566
	Nigeria CountryCode = 566
	// Niue                                   CountryCode = 570
	Niue CountryCode = 570
	// NorfolkIsland                          CountryCode = 574
	NorfolkIsland CountryCode = 574
	// NorthernMarianaIslands                 CountryCode = 580
	NorthernMarianaIslands CountryCode = 580
	// Norway                                 CountryCode = 578
	Norway CountryCode = 578
	// Oman                                   CountryCode = 512
	Oman CountryCode = 512
	// Pakistan                               CountryCode = 586
	Pakistan CountryCode = 586
	// Palau                                  CountryCode = 585
	Palau CountryCode = 585
	// Palestine                              CountryCode = 275
	Palestine CountryCode = 275
	// Panama                                 CountryCode = 591
	Panama CountryCode = 591
	// PapuaNewGuinea                         CountryCode = 598
	PapuaNewGuinea CountryCode = 598
	// Paraguay                               CountryCode = 600
	Paraguay CountryCode = 600
	// Peru                                   CountryCode = 604
	Peru CountryCode = 604
	// Philippines                            CountryCode = 608
	Philippines CountryCode = 608
	// Pitcairn                               CountryCode = 612
	Pitcairn CountryCode = 612
	// Poland                                 CountryCode = 616
	Poland CountryCode = 616
	// Portugal                               CountryCode = 620
	Portugal CountryCode = 620
	// PuertoRico                             CountryCode = 630
	PuertoRico CountryCode = 630
	// Qatar                                  CountryCode = 634
	Qatar CountryCode = 634
	// Reunion                                CountryCode = 638
	Reunion CountryCode = 638
	// Romania                                CountryCode = 642
	Romania CountryCode = 642
	// Russia                                 CountryCode = 643
	Russia CountryCode = 643
	// Rwanda                                 CountryCode = 646
	Rwanda CountryCode = 646
	// SaintHelena                            CountryCode = 654
	SaintHelena CountryCode = 654
	// SaintKittsAndNevis                     CountryCode = 659
	SaintKittsAndNevis CountryCode = 659
	// SaintLucia                             CountryCode = 662
	SaintLucia CountryCode = 662
	// SaintPierreAndMiquelon                 CountryCode = 666
	SaintPierreAndMiquelon CountryCode = 666
	// SaintVincentAndTheGrenadines           CountryCode = 670
	SaintVincentAndTheGrenadines CountryCode = 670
	// Samoa                                  CountryCode = 882
	Samoa CountryCode = 882
	// SanMarino                              CountryCode = 674
	SanMarino CountryCode = 674
	// SaoTomeAndPrincipe                     CountryCode = 678
	SaoTomeAndPrincipe CountryCode = 678
	// SaudiArabia                            CountryCode = 682
	SaudiArabia CountryCode = 682
	// Senegal                                CountryCode = 686
	Senegal CountryCode = 686
	// Seychelles                             CountryCode = 690
	Seychelles CountryCode = 690
	// SierraLeone                            CountryCode = 694
	SierraLeone CountryCode = 694
	// Singapore                              CountryCode = 702
	Singapore CountryCode = 702
	// Slovakia                               CountryCode = 703
	Slovakia CountryCode = 703
	// Slovenia                               CountryCode = 705
	Slovenia CountryCode = 705
	// SolomonIslands                         CountryCode = 90
	SolomonIslands CountryCode = 90
	// Somalia                                CountryCode = 706
	Somalia CountryCode = 706
	// SouthAfrica                            CountryCode = 710
	SouthAfrica CountryCode = 710
	// UAR                                    CountryCode = 710
	UAR CountryCode = 710
	// SouthGeorgiaAndTheSouthSandwichIslands CountryCode = 239
	SouthGeorgiaAndTheSouthSandwichIslands CountryCode = 239
	// Spain                                  CountryCode = 724
	Spain CountryCode = 724
	// SriLanka                               CountryCode = 144
	SriLanka CountryCode = 144
	// Sudan                                  CountryCode = 729
	Sudan CountryCode = 729
	// Suriname                               CountryCode = 740
	Suriname CountryCode = 740
	// SvalbardAndJanMayenIslands             CountryCode = 744
	SvalbardAndJanMayenIslands CountryCode = 744
	// Swaziland                              CountryCode = 748
	Swaziland CountryCode = 748
	// Sweden                                 CountryCode = 752
	Sweden CountryCode = 752
	// Scotland                               CountryCode = 826
	Scotland CountryCode = 826
	// Switzerland                            CountryCode = 756
	Switzerland CountryCode = 756
	// Syria                                  CountryCode = 760
	Syria CountryCode = 760
	// Taiwan                                 CountryCode = 158
	Taiwan CountryCode = 158
	// Tajikistan                             CountryCode = 762
	Tajikistan CountryCode = 762
	// Tanzania                               CountryCode = 834
	Tanzania CountryCode = 834
	// Thailand                               CountryCode = 764
	Thailand CountryCode = 764
	// TimorLeste                             CountryCode = 626
	TimorLeste CountryCode = 626
	// Togo                                   CountryCode = 768
	Togo CountryCode = 768
	// Tokelau                                CountryCode = 772
	Tokelau CountryCode = 772
	// Tonga                                  CountryCode = 776
	Tonga CountryCode = 776
	// TrinidadAndTobago                      CountryCode = 780
	TrinidadAndTobago CountryCode = 780
	// Tunisia                                CountryCode = 788
	Tunisia CountryCode = 788
	// Turkey                                 CountryCode = 792
	Turkey CountryCode = 792
	// Turkmenistan                           CountryCode = 795
	Turkmenistan CountryCode = 795
	// TurksAndCaicosIslands                  CountryCode = 796
	TurksAndCaicosIslands CountryCode = 796
	// Tuvalu                                 CountryCode = 798
	Tuvalu CountryCode = 798
	// Uganda                                 CountryCode = 800
	Uganda CountryCode = 800
	// Ukraine                                CountryCode = 804
	Ukraine CountryCode = 804
	// UnitedArabEmirates                     CountryCode = 784
	UnitedArabEmirates CountryCode = 784
	// UnitedKingdom                          CountryCode = 826
	UnitedKingdom CountryCode = 826
	// UnitedStatesOfAmerica                  CountryCode = 840
	UnitedStatesOfAmerica CountryCode = 840
	// UnitedStatesMinorOutlyingIslands       CountryCode = 581
	UnitedStatesMinorOutlyingIslands CountryCode = 581
	// Uruguay                                CountryCode = 858
	Uruguay CountryCode = 858
	// Wales                                  CountryCode = 826
	Wales CountryCode = 826
	// Uzbekistan                             CountryCode = 860
	Uzbekistan CountryCode = 860
	// Vanuatu                                CountryCode = 548
	Vanuatu CountryCode = 548
	// HolySee                                CountryCode = 336
	HolySee CountryCode = 336
	// Venezuela                              CountryCode = 862
	Venezuela CountryCode = 862
	// Vietnam                                CountryCode = 704
	Vietnam CountryCode = 704
	// VirginIslandsBritish                   CountryCode = 92
	VirginIslandsBritish CountryCode = 92
	// VirginIslandsUS                        CountryCode = 850
	VirginIslandsUS CountryCode = 850
	// WallisandFutunaIslands                 CountryCode = 876
	WallisandFutunaIslands CountryCode = 876
	// WesternSahara                          CountryCode = 732
	WesternSahara CountryCode = 732
	// Yemen                                  CountryCode = 887
	Yemen CountryCode = 887
	// Yugoslavia                             CountryCode = 891
	Yugoslavia CountryCode = 891
	// Zambia                                 CountryCode = 894
	Zambia CountryCode = 894
	// Zimbabwe                               CountryCode = 716
	Zimbabwe CountryCode = 716
	// Afghanistan                            CountryCode = 4
	Afghanistan CountryCode = 4
	// Serbia                                 CountryCode = 688
	Serbia CountryCode = 688
	// AlandIslands                           CountryCode = 248
	AlandIslands CountryCode = 248
	// Bonaire                                CountryCode = 535
	Bonaire CountryCode = 535
	// Guernsey                               CountryCode = 831
	Guernsey CountryCode = 831
	// Jersey                                 CountryCode = 832
	Jersey CountryCode = 832
	// Curacao                                CountryCode = 531
	Curacao CountryCode = 531
	// SaintBarthelemy                        CountryCode = 652
	SaintBarthelemy CountryCode = 652
	// SaintMartinFrench                      CountryCode = 663
	SaintMartinFrench CountryCode = 663
	// SintMaartenDutch                       CountryCode = 534
	SintMaartenDutch CountryCode = 534
	// Montenegro                             CountryCode = 499
	Montenegro CountryCode = 499
	// SouthSudan                             CountryCode = 728
	SouthSudan CountryCode = 728
	// Kosovo                                 CountryCode = 900
	Kosovo CountryCode = 900
	// None									  CountryCode = 998
	None CountryCode = 998
)

// Non-countries codes
const (
	// NonCountryInternationalFreephone                               CountryCode = 999800
	NonCountryInternationalFreephone CountryCode = 999800 // for callcode +800, International Freephone (UIFN)
	// NonCountryInmarsat                                             CountryCode = 999870
	NonCountryInmarsat CountryCode = 999870 // for callcode +870, Inmarsat "SNAC" service
	// NonCountryMaritimeMobileService                                CountryCode = 999875
	NonCountryMaritimeMobileService CountryCode = 999875 // for callcodes +875, +876, +877
	// NonCountryUniversalPersonalTelecommunicationsServices          CountryCode = 999878
	NonCountryUniversalPersonalTelecommunicationsServices CountryCode = 999878 // for callcode +878
	// NonCountryNationalNonCommercialPurposes                        CountryCode = 999879
	NonCountryNationalNonCommercialPurposes CountryCode = 999879 // for callcode +879
	// NonCountryGlobalMobileSatelliteSystem                          CountryCode = 999881
	NonCountryGlobalMobileSatelliteSystem CountryCode = 999881 // for callcode +881
	// NonCountryInternationalNetworks                                CountryCode = 999882
	NonCountryInternationalNetworks CountryCode = 999882 // for callcodes +882, +883
	// NonCountryDisasterRelief                                       CountryCode = 999888
	NonCountryDisasterRelief CountryCode = 999888 // for callcode +888
	// NonCountryInternationalPremiumRateService                      CountryCode = 999979
	NonCountryInternationalPremiumRateService CountryCode = 999979 // for callcode +979
	// NonCountryInternationalTelecommunicationsCorrespondenceService CountryCode = 999991
	NonCountryInternationalTelecommunicationsCorrespondenceService CountryCode = 999991 // for callcode +991
)

// textPrepare prepares text for comparison by converting to uppercase and removing spaces

// NOTE: it works very more faster than strings.Replacer and regexp.Regexp
func textPrepare(text string) string {
	indx := strings.Index(text, "(")
	if indx > -1 {
		text = text[:indx]
	}

	reader := strings.NewReader(text)
	text = ""

	var r rune
	var err error
	for {
		r, _, err = reader.ReadRune()
		if err == io.EOF {
			break
		}
		if unicode.IsLetter(r) {
			text += string(r)
		}
	}

	return strings.ToUpper(text)
}

// Alpha-2 digit ISO 3166-1. Three codes present, for example Russia == RU == RUS == 643.
const (
	// AL CountryCode = 8
	AL CountryCode = 8
	// DZ CountryCode = 12
	DZ CountryCode = 12
	// AS CountryCode = 16
	AS CountryCode = 16
	// AD CountryCode = 20
	AD CountryCode = 20
	// AO CountryCode = 24
	AO CountryCode = 24
	// AI CountryCode = 660
	AI CountryCode = 660
	// AQ CountryCode = 10
	AQ CountryCode = 10
	// AG CountryCode = 28
	AG CountryCode = 28
	// AR CountryCode = 32
	AR CountryCode = 32
	// AM CountryCode = 51
	AM CountryCode = 51
	// AW CountryCode = 533
	AW CountryCode = 533
	// AU CountryCode = 36
	AU CountryCode = 36
	// AT CountryCode = 40
	AT CountryCode = 40
	// AZ CountryCode = 31
	AZ CountryCode = 31
	// BS CountryCode = 44
	BS CountryCode = 44
	// BH CountryCode = 48
	BH CountryCode = 48
	// BD CountryCode = 50
	BD CountryCode = 50
	// BB CountryCode = 52
	BB CountryCode = 52
	// BY CountryCode = 112
	BY CountryCode = 112
	// BE CountryCode = 56
	BE CountryCode = 56
	// BZ CountryCode = 84
	BZ CountryCode = 84
	// BJ CountryCode = 204
	BJ CountryCode = 204
	// BM CountryCode = 60
	BM CountryCode = 60
	// BT CountryCode = 64
	BT CountryCode = 64
	// BO CountryCode = 68
	BO CountryCode = 68
	// BA CountryCode = 70
	BA CountryCode = 70
	// BW CountryCode = 72
	BW CountryCode = 72
	// BV CountryCode = 74
	BV CountryCode = 74
	// BR CountryCode = 76
	BR CountryCode = 76
	// IO CountryCode = 86
	IO CountryCode = 86
	// BN CountryCode = 96
	BN CountryCode = 96
	// BG CountryCode = 100
	BG CountryCode = 100
	// BF CountryCode = 854
	BF CountryCode = 854
	// BI CountryCode = 108
	BI CountryCode = 108
	// KH CountryCode = 116
	KH CountryCode = 116
	// CM CountryCode = 120
	CM CountryCode = 120
	// CA CountryCode = 124
	CA CountryCode = 124
	// CV CountryCode = 132
	CV CountryCode = 132
	// KY CountryCode = 136
	KY CountryCode = 136
	// CF CountryCode = 140
	CF CountryCode = 140
	// TD CountryCode = 148
	TD CountryCode = 148
	// CL CountryCode = 152
	CL CountryCode = 152
	// CN CountryCode = 156
	CN CountryCode = 156
	// CX CountryCode = 162
	CX CountryCode = 162
	// CC CountryCode = 166
	CC CountryCode = 166
	// CO CountryCode = 170
	CO CountryCode = 170
	// KM CountryCode = 174
	KM CountryCode = 174
	// CG CountryCode = 178
	CG CountryCode = 178
	// CD CountryCode = 180
	CD CountryCode = 180
	// CK CountryCode = 184
	CK CountryCode = 184
	// CR CountryCode = 188
	CR CountryCode = 188
	// CI CountryCode = 384
	CI CountryCode = 384
	// HR CountryCode = 191
	HR CountryCode = 191
	// CU CountryCode = 192
	CU CountryCode = 192
	// CY CountryCode = 196
	CY CountryCode = 196
	// CZ CountryCode = 203
	CZ CountryCode = 203
	// DK CountryCode = 208
	DK CountryCode = 208
	// DJ CountryCode = 262
	DJ CountryCode = 262
	// DM CountryCode = 212
	DM CountryCode = 212
	// DO CountryCode = 214
	DO CountryCode = 214
	// EC CountryCode = 218
	EC CountryCode = 218
	// EG CountryCode = 818
	EG CountryCode = 818
	// SV CountryCode = 222
	SV CountryCode = 222
	// GQ CountryCode = 226
	GQ CountryCode = 226
	// ER CountryCode = 232
	ER CountryCode = 232
	// EE CountryCode = 233
	EE CountryCode = 233
	// ET CountryCode = 231
	ET CountryCode = 231
	// FO CountryCode = 234
	FO CountryCode = 234
	// FK CountryCode = 238
	FK CountryCode = 238
	// FJ CountryCode = 242
	FJ CountryCode = 242
	// FI CountryCode = 246
	FI CountryCode = 246
	// FR CountryCode = 250
	FR CountryCode = 250
	// GF CountryCode = 254
	GF CountryCode = 254
	// PF CountryCode = 258
	PF CountryCode = 258
	// TF CountryCode = 260
	TF CountryCode = 260
	// GA CountryCode = 266
	GA CountryCode = 266
	// GM CountryCode = 270
	GM CountryCode = 270
	// GE CountryCode = 268
	GE CountryCode = 268
	// DE CountryCode = 276
	DE CountryCode = 276
	// GH CountryCode = 288
	GH CountryCode = 288
	// GI CountryCode = 292
	GI CountryCode = 292
	// GR CountryCode = 300
	GR CountryCode = 300
	// GL CountryCode = 304
	GL CountryCode = 304
	// GD CountryCode = 308
	GD CountryCode = 308
	// GP CountryCode = 312
	GP CountryCode = 312
	// GU CountryCode = 316
	GU CountryCode = 316
	// GT CountryCode = 320
	GT CountryCode = 320
	// GN CountryCode = 324
	GN CountryCode = 324
	// GW CountryCode = 624
	GW CountryCode = 624
	// GY CountryCode = 328
	GY CountryCode = 328
	// HT CountryCode = 332
	HT CountryCode = 332
	// HM CountryCode = 334
	HM CountryCode = 334
	// HN CountryCode = 340
	HN CountryCode = 340
	// HK CountryCode = 344
	HK CountryCode = 344
	// HU CountryCode = 348
	HU CountryCode = 348
	// IS CountryCode = 352
	IS CountryCode = 352
	// IN CountryCode = 356
	IN CountryCode = 356
	// ID CountryCode = 360
	ID CountryCode = 360
	// IR CountryCode = 364
	IR CountryCode = 364
	// IQ CountryCode = 368
	IQ CountryCode = 368
	// IE CountryCode = 372
	IE CountryCode = 372
	// IL CountryCode = 376
	IL CountryCode = 376
	// IT CountryCode = 380
	IT CountryCode = 380
	// JM CountryCode = 388
	JM CountryCode = 388
	// JP CountryCode = 392
	JP CountryCode = 392
	// JO CountryCode = 400
	JO CountryCode = 400
	// KZ CountryCode = 398
	KZ CountryCode = 398
	// KE CountryCode = 404
	KE CountryCode = 404
	// KI CountryCode = 296
	KI CountryCode = 296
	// KR CountryCode = 410
	KR CountryCode = 410
	// KP CountryCode = 408
	KP CountryCode = 408
	// KW CountryCode = 414
	KW CountryCode = 414
	// KG CountryCode = 417
	KG CountryCode = 417
	// LA CountryCode = 418
	LA CountryCode = 418
	// LV CountryCode = 428
	LV CountryCode = 428
	// LB CountryCode = 422
	LB CountryCode = 422
	// LS CountryCode = 426
	LS CountryCode = 426
	// LR CountryCode = 430
	LR CountryCode = 430
	// LY CountryCode = 434
	LY CountryCode = 434
	// LI CountryCode = 438
	LI CountryCode = 438
	// LT CountryCode = 440
	LT CountryCode = 440
	// LU CountryCode = 442
	LU CountryCode = 442
	// MO CountryCode = 446
	MO CountryCode = 446
	// MK CountryCode = 807
	MK CountryCode = 807
	// MG CountryCode = 450
	MG CountryCode = 450
	// MW CountryCode = 454
	MW CountryCode = 454
	// MY CountryCode = 458
	MY CountryCode = 458
	// MV CountryCode = 462
	MV CountryCode = 462
	// ML CountryCode = 466
	ML CountryCode = 466
	// MT CountryCode = 470
	MT CountryCode = 470
	// MH CountryCode = 584
	MH CountryCode = 584
	// MQ CountryCode = 474
	MQ CountryCode = 474
	// MR CountryCode = 478
	MR CountryCode = 478
	// MU CountryCode = 480
	MU CountryCode = 480
	// YT CountryCode = 175
	YT CountryCode = 175
	// MX CountryCode = 484
	MX CountryCode = 484
	// FM CountryCode = 583
	FM CountryCode = 583
	// MD CountryCode = 498
	MD CountryCode = 498
	// MC CountryCode = 492
	MC CountryCode = 492
	// MN CountryCode = 496
	MN CountryCode = 496
	// MS CountryCode = 500
	MS CountryCode = 500
	// MA CountryCode = 504
	MA CountryCode = 504
	// MZ CountryCode = 508
	MZ CountryCode = 508
	// MM CountryCode = 104
	MM CountryCode = 104
	// NA CountryCode = 516
	NA CountryCode = 516
	// NR CountryCode = 520
	NR CountryCode = 520
	// NP CountryCode = 524
	NP CountryCode = 524
	// NL CountryCode = 528
	NL CountryCode = 528
	// AN CountryCode = 530
	AN CountryCode = 530
	// NC CountryCode = 540
	NC CountryCode = 540
	// NZ CountryCode = 554
	NZ CountryCode = 554
	// NI CountryCode = 558
	NI CountryCode = 558
	// NE CountryCode = 562
	NE CountryCode = 562
	// NG CountryCode = 566
	NG CountryCode = 566
	// NU CountryCode = 570
	NU CountryCode = 570
	// NF CountryCode = 574
	NF CountryCode = 574
	// MP CountryCode = 580
	MP CountryCode = 580
	// NO CountryCode = 578
	NO CountryCode = 578
	// OM CountryCode = 512
	OM CountryCode = 512
	// PK CountryCode = 586
	PK CountryCode = 586
	// PW CountryCode = 585
	PW CountryCode = 585
	// PS CountryCode = 275
	PS CountryCode = 275
	// PA CountryCode = 591
	PA CountryCode = 591
	// PG CountryCode = 598
	PG CountryCode = 598
	// PY CountryCode = 600
	PY CountryCode = 600
	// PE CountryCode = 604
	PE CountryCode = 604
	// PH CountryCode = 608
	PH CountryCode = 608
	// PN CountryCode = 612
	PN CountryCode = 612
	// PL CountryCode = 616
	PL CountryCode = 616
	// PT CountryCode = 620
	PT CountryCode = 620
	// PR CountryCode = 630
	PR CountryCode = 630
	// QA CountryCode = 634
	QA CountryCode = 634
	// RE CountryCode = 638
	RE CountryCode = 638
	// RO CountryCode = 642
	RO CountryCode = 642
	// RU CountryCode = 643
	RU CountryCode = 643
	// RW CountryCode = 646
	RW CountryCode = 646
	// SH CountryCode = 654
	SH CountryCode = 654
	// KN CountryCode = 659
	KN CountryCode = 659
	// LC CountryCode = 662
	LC CountryCode = 662
	// PM CountryCode = 666
	PM CountryCode = 666
	// VC CountryCode = 670
	VC CountryCode = 670
	// WS CountryCode = 882
	WS CountryCode = 882
	// SM CountryCode = 674
	SM CountryCode = 674
	// ST CountryCode = 678
	ST CountryCode = 678
	// SA CountryCode = 682
	SA CountryCode = 682
	// SN CountryCode = 686
	SN CountryCode = 686
	// SC CountryCode = 690
	SC CountryCode = 690
	// SL CountryCode = 694
	SL CountryCode = 694
	// SG CountryCode = 702
	SG CountryCode = 702
	// SK CountryCode = 703
	SK CountryCode = 703
	// SI CountryCode = 705
	SI CountryCode = 705
	// SB CountryCode = 90
	SB CountryCode = 90
	// SO CountryCode = 706
	SO CountryCode = 706
	// ZA CountryCode = 710
	ZA CountryCode = 710
	// GS CountryCode = 239
	GS CountryCode = 239
	// ES CountryCode = 724
	ES CountryCode = 724
	// LK CountryCode = 144
	LK CountryCode = 144
	// SD CountryCode = 729
	SD CountryCode = 729
	// SR CountryCode = 740
	SR CountryCode = 740
	// SJ CountryCode = 744
	SJ CountryCode = 744
	// SZ CountryCode = 748
	SZ CountryCode = 748
	// SE CountryCode = 752
	SE CountryCode = 752
	// XS CountryCode = 826
	XS CountryCode = 826
	// CH CountryCode = 756
	CH CountryCode = 756
	// SY CountryCode = 760
	SY CountryCode = 760
	// TW CountryCode = 158
	TW CountryCode = 158
	// TJ CountryCode = 762
	TJ CountryCode = 762
	// TZ CountryCode = 834
	TZ CountryCode = 834
	// TH CountryCode = 764
	TH CountryCode = 764
	// TL CountryCode = 626
	TL CountryCode = 626
	// TG CountryCode = 768
	TG CountryCode = 768
	// TK CountryCode = 772
	TK CountryCode = 772
	// TO CountryCode = 776
	TO CountryCode = 776
	// TT CountryCode = 780
	TT CountryCode = 780
	// TN CountryCode = 788
	TN CountryCode = 788
	// TR CountryCode = 792
	TR CountryCode = 792
	// TM CountryCode = 795
	TM CountryCode = 795
	// TC CountryCode = 796
	TC CountryCode = 796
	// TV CountryCode = 798
	TV CountryCode = 798
	// UG CountryCode = 800
	UG CountryCode = 800
	// UA CountryCode = 804
	UA CountryCode = 804
	// AE CountryCode = 784
	AE CountryCode = 784
	// GB CountryCode = 826
	GB CountryCode = 826
	// US CountryCode = 840
	US CountryCode = 840
	// UM CountryCode = 581
	UM CountryCode = 581
	// UY CountryCode = 858
	UY CountryCode = 858
	// UZ CountryCode = 860
	UZ CountryCode = 860
	// VU CountryCode = 548
	VU CountryCode = 548
	// VA CountryCode = 336
	VA CountryCode = 336
	// VE CountryCode = 862
	VE CountryCode = 862
	// VN CountryCode = 704
	VN CountryCode = 704
	// VG CountryCode = 92
	VG CountryCode = 92
	// VI CountryCode = 850
	VI CountryCode = 850
	// WF CountryCode = 876
	WF CountryCode = 876
	// EH CountryCode = 732
	EH CountryCode = 732
	// YE CountryCode = 887
	YE CountryCode = 887
	// YU CountryCode = 891
	YU CountryCode = 891
	// ZM CountryCode = 894
	ZM CountryCode = 894
	// ZW CountryCode = 716
	ZW CountryCode = 716
	// AF CountryCode = 4
	AF CountryCode = 4
	// RS CountryCode = 688
	RS CountryCode = 688
	// AX CountryCode = 248
	AX CountryCode = 248
	// BQ CountryCode = 535
	BQ CountryCode = 535
	// GG CountryCode = 831
	GG CountryCode = 831
	// JE CountryCode = 832
	JE CountryCode = 832
	// CW CountryCode = 531
	CW CountryCode = 531
	// IM CountryCode = 833
	IM CountryCode = 833
	// BL CountryCode = 652
	BL CountryCode = 652
	// MF CountryCode = 663
	MF CountryCode = 663
	// SX CountryCode = 534
	SX CountryCode = 534
	// ME CountryCode = 499
	ME CountryCode = 499
	// SS CountryCode = 728
	SS CountryCode = 728
	// XK CountryCode = 900
	XK CountryCode = 900
	// XX CountryCode = 998
	XX CountryCode = 998
)

// Alpha-3 digit ISO 3166-1. Three codes present, for example Russia == RU == RUS == 643.
const (
	// ALB CountryCode = 8
	ALB CountryCode = 8
	// DZA CountryCode = 12
	DZA CountryCode = 12
	// ASM CountryCode = 16
	ASM CountryCode = 16
	// AND CountryCode = 20
	AND CountryCode = 20
	// AGO CountryCode = 24
	AGO CountryCode = 24
	// AIA CountryCode = 660
	AIA CountryCode = 660
	// ATA CountryCode = 10
	ATA CountryCode = 10
	// ATG CountryCode = 28
	ATG CountryCode = 28
	// ARG CountryCode = 32
	ARG CountryCode = 32
	// ARM CountryCode = 51
	ARM CountryCode = 51
	// ABW CountryCode = 533
	ABW CountryCode = 533
	// AUS CountryCode = 36
	AUS CountryCode = 36
	// AUT CountryCode = 40
	AUT CountryCode = 40
	// AZE CountryCode = 31
	AZE CountryCode = 31
	// BHS CountryCode = 44
	BHS CountryCode = 44
	// BHR CountryCode = 48
	BHR CountryCode = 48
	// BGD CountryCode = 50
	BGD CountryCode = 50
	// BRB CountryCode = 52
	BRB CountryCode = 52
	// BLR CountryCode = 112
	BLR CountryCode = 112
	// BEL CountryCode = 56
	BEL CountryCode = 56
	// BLZ CountryCode = 84
	BLZ CountryCode = 84
	// BEN CountryCode = 204
	BEN CountryCode = 204
	// BMU CountryCode = 60
	BMU CountryCode = 60
	// BTN CountryCode = 64
	BTN CountryCode = 64
	// BOL CountryCode = 68
	BOL CountryCode = 68
	// BIH CountryCode = 70
	BIH CountryCode = 70
	// BWA CountryCode = 72
	BWA CountryCode = 72
	// BVT CountryCode = 74
	BVT CountryCode = 74
	// BRA CountryCode = 76
	BRA CountryCode = 76
	// IOT CountryCode = 86
	IOT CountryCode = 86
	// BRN CountryCode = 96
	BRN CountryCode = 96
	// BGR CountryCode = 100
	BGR CountryCode = 100
	// BFA CountryCode = 854
	BFA CountryCode = 854
	// BDI CountryCode = 108
	BDI CountryCode = 108
	// KHM CountryCode = 116
	KHM CountryCode = 116
	// CMR CountryCode = 120
	CMR CountryCode = 120
	// CAN CountryCode = 124
	CAN CountryCode = 124
	// CPV CountryCode = 132
	CPV CountryCode = 132
	// CYM CountryCode = 136
	CYM CountryCode = 136
	// CAF CountryCode = 140
	CAF CountryCode = 140
	// TCD CountryCode = 148
	TCD CountryCode = 148
	// CHL CountryCode = 152
	CHL CountryCode = 152
	// CHN CountryCode = 156
	CHN CountryCode = 156
	// CXR CountryCode = 162
	CXR CountryCode = 162
	// CCK CountryCode = 166
	CCK CountryCode = 166
	// COL CountryCode = 170
	COL CountryCode = 170
	// COM CountryCode = 174
	COM CountryCode = 174
	// COG CountryCode = 178
	COG CountryCode = 178
	// COD CountryCode = 180
	COD CountryCode = 180
	// COK CountryCode = 184
	COK CountryCode = 184
	// CRI CountryCode = 188
	CRI CountryCode = 188
	// CIV CountryCode = 384
	CIV CountryCode = 384
	// HRV CountryCode = 191
	HRV CountryCode = 191
	// CUB CountryCode = 192
	CUB CountryCode = 192
	// CYP CountryCode = 196
	CYP CountryCode = 196
	// CZE CountryCode = 203
	CZE CountryCode = 203
	// DNK CountryCode = 208
	DNK CountryCode = 208
	// DJI CountryCode = 262
	DJI CountryCode = 262
	// DMA CountryCode = 212
	DMA CountryCode = 212
	// DOM CountryCode = 214
	DOM CountryCode = 214
	// ECU CountryCode = 218
	ECU CountryCode = 218
	// EGY CountryCode = 818
	EGY CountryCode = 818
	// SLV CountryCode = 222
	SLV CountryCode = 222
	// GNQ CountryCode = 226
	GNQ CountryCode = 226
	// ERI CountryCode = 232
	ERI CountryCode = 232
	// EST CountryCode = 233
	EST CountryCode = 233
	// ETH CountryCode = 231
	ETH CountryCode = 231
	// FRO CountryCode = 234
	FRO CountryCode = 234
	// FLK CountryCode = 238
	FLK CountryCode = 238
	// FJI CountryCode = 242
	FJI CountryCode = 242
	// FIN CountryCode = 246
	FIN CountryCode = 246
	// FRA CountryCode = 250
	FRA CountryCode = 250
	// GUF CountryCode = 254
	GUF CountryCode = 254
	// PYF CountryCode = 258
	PYF CountryCode = 258
	// ATF CountryCode = 260
	ATF CountryCode = 260
	// GAB CountryCode = 266
	GAB CountryCode = 266
	// GMB CountryCode = 270
	GMB CountryCode = 270
	// GEO CountryCode = 268
	GEO CountryCode = 268
	// DEU CountryCode = 276
	DEU CountryCode = 276
	// GHA CountryCode = 288
	GHA CountryCode = 288
	// GIB CountryCode = 292
	GIB CountryCode = 292
	// GRC CountryCode = 300
	GRC CountryCode = 300
	// GRL CountryCode = 304
	GRL CountryCode = 304
	// GRD CountryCode = 308
	GRD CountryCode = 308
	// GLP CountryCode = 312
	GLP CountryCode = 312
	// GUM CountryCode = 316
	GUM CountryCode = 316
	// GTM CountryCode = 320
	GTM CountryCode = 320
	// GIN CountryCode = 324
	GIN CountryCode = 324
	// GNB CountryCode = 624
	GNB CountryCode = 624
	// GUY CountryCode = 328
	GUY CountryCode = 328
	// HTI CountryCode = 332
	HTI CountryCode = 332
	// HMD CountryCode = 334
	HMD CountryCode = 334
	// HND CountryCode = 340
	HND CountryCode = 340
	// HKG CountryCode = 344
	HKG CountryCode = 344
	// HUN CountryCode = 348
	HUN CountryCode = 348
	// ISL CountryCode = 352
	ISL CountryCode = 352
	// IND CountryCode = 356
	IND CountryCode = 356
	// IDN CountryCode = 360
	IDN CountryCode = 360
	// IRN CountryCode = 364
	IRN CountryCode = 364
	// IRQ CountryCode = 368
	IRQ CountryCode = 368
	// IRL CountryCode = 372
	IRL CountryCode = 372
	// ISR CountryCode = 376
	ISR CountryCode = 376
	// ITA CountryCode = 380
	ITA CountryCode = 380
	// JAM CountryCode = 388
	JAM CountryCode = 388
	// JPN CountryCode = 392
	JPN CountryCode = 392
	// JOR CountryCode = 400
	JOR CountryCode = 400
	// KAZ CountryCode = 398
	KAZ CountryCode = 398
	// KEN CountryCode = 404
	KEN CountryCode = 404
	// KIR CountryCode = 296
	KIR CountryCode = 296
	// KOR CountryCode = 410
	KOR CountryCode = 410
	// PRK CountryCode = 408
	PRK CountryCode = 408
	// KWT CountryCode = 414
	KWT CountryCode = 414
	// KGZ CountryCode = 417
	KGZ CountryCode = 417
	// LAO CountryCode = 418
	LAO CountryCode = 418
	// LVA CountryCode = 428
	LVA CountryCode = 428
	// LBN CountryCode = 422
	LBN CountryCode = 422
	// LSO CountryCode = 426
	LSO CountryCode = 426
	// LBR CountryCode = 430
	LBR CountryCode = 430
	// LBY CountryCode = 434
	LBY CountryCode = 434
	// LIE CountryCode = 438
	LIE CountryCode = 438
	// LTU CountryCode = 440
	LTU CountryCode = 440
	// LUX CountryCode = 442
	LUX CountryCode = 442
	// MAC CountryCode = 446
	MAC CountryCode = 446
	// MKD CountryCode = 807
	MKD CountryCode = 807
	// MDG CountryCode = 450
	MDG CountryCode = 450
	// MWI CountryCode = 454
	MWI CountryCode = 454
	// MYS CountryCode = 458
	MYS CountryCode = 458
	// MDV CountryCode = 462
	MDV CountryCode = 462
	// MLI CountryCode = 466
	MLI CountryCode = 466
	// MLT CountryCode = 470
	MLT CountryCode = 470
	// MHL CountryCode = 584
	MHL CountryCode = 584
	// MTQ CountryCode = 474
	MTQ CountryCode = 474
	// MRT CountryCode = 478
	MRT CountryCode = 478
	// MUS CountryCode = 480
	MUS CountryCode = 480
	// MYT CountryCode = 175
	MYT CountryCode = 175
	// MEX CountryCode = 484
	MEX CountryCode = 484
	// FSM CountryCode = 583
	FSM CountryCode = 583
	// MDA CountryCode = 498
	MDA CountryCode = 498
	// MCO CountryCode = 492
	MCO CountryCode = 492
	// MNG CountryCode = 496
	MNG CountryCode = 496
	// MSR CountryCode = 500
	MSR CountryCode = 500
	// MAR CountryCode = 504
	MAR CountryCode = 504
	// MOZ CountryCode = 508
	MOZ CountryCode = 508
	// MMR CountryCode = 104
	MMR CountryCode = 104
	// NAM CountryCode = 516
	NAM CountryCode = 516
	// NRU CountryCode = 520
	NRU CountryCode = 520
	// NPL CountryCode = 524
	NPL CountryCode = 524
	// NLD CountryCode = 528
	NLD CountryCode = 528
	// ANT CountryCode = 530
	ANT CountryCode = 530
	// NCL CountryCode = 540
	NCL CountryCode = 540
	// NZL CountryCode = 554
	NZL CountryCode = 554
	// NIC CountryCode = 558
	NIC CountryCode = 558
	// NER CountryCode = 562
	NER CountryCode = 562
	// NGA CountryCode = 566
	NGA CountryCode = 566
	// NIU CountryCode = 570
	NIU CountryCode = 570
	// NFK CountryCode = 574
	NFK CountryCode = 574
	// MNP CountryCode = 580
	MNP CountryCode = 580
	// NOR CountryCode = 578
	NOR CountryCode = 578
	// OMN CountryCode = 512
	OMN CountryCode = 512
	// PAK CountryCode = 586
	PAK CountryCode = 586
	// PLW CountryCode = 585
	PLW CountryCode = 585
	// PSE CountryCode = 275
	PSE CountryCode = 275
	// PAN CountryCode = 591
	PAN CountryCode = 591
	// PNG CountryCode = 598
	PNG CountryCode = 598
	// PRY CountryCode = 600
	PRY CountryCode = 600
	// PER CountryCode = 604
	PER CountryCode = 604
	// PHL CountryCode = 608
	PHL CountryCode = 608
	// PCN CountryCode = 612
	PCN CountryCode = 612
	// POL CountryCode = 616
	POL CountryCode = 616
	// PRT CountryCode = 620
	PRT CountryCode = 620
	// PRI CountryCode = 630
	PRI CountryCode = 630
	// QAT CountryCode = 634
	QAT CountryCode = 634
	// REU CountryCode = 638
	REU CountryCode = 638
	// ROU CountryCode = 642
	ROU CountryCode = 642
	// RUS CountryCode = 643
	RUS CountryCode = 643
	// RWA CountryCode = 646
	RWA CountryCode = 646
	// SHN CountryCode = 654
	SHN CountryCode = 654
	// KNA CountryCode = 659
	KNA CountryCode = 659
	// LCA CountryCode = 662
	LCA CountryCode = 662
	// SPM CountryCode = 666
	SPM CountryCode = 666
	// VCT CountryCode = 670
	VCT CountryCode = 670
	// WSM CountryCode = 882
	WSM CountryCode = 882
	// SMR CountryCode = 674
	SMR CountryCode = 674
	// STP CountryCode = 678
	STP CountryCode = 678
	// SAU CountryCode = 682
	SAU CountryCode = 682
	// SEN CountryCode = 686
	SEN CountryCode = 686
	// SYC CountryCode = 690
	SYC CountryCode = 690
	// SLE CountryCode = 694
	SLE CountryCode = 694
	// SGP CountryCode = 702
	SGP CountryCode = 702
	// SVK CountryCode = 703
	SVK CountryCode = 703
	// SVN CountryCode = 705
	SVN CountryCode = 705
	// SLB CountryCode = 90
	SLB CountryCode = 90
	// SOM CountryCode = 706
	SOM CountryCode = 706
	// ZAF CountryCode = 710
	ZAF CountryCode = 710
	// SGS CountryCode = 239
	SGS CountryCode = 239
	// ESP CountryCode = 724
	ESP CountryCode = 724
	// LKA CountryCode = 144
	LKA CountryCode = 144
	// SDN CountryCode = 729
	SDN CountryCode = 729
	// SUR CountryCode = 740
	SUR CountryCode = 740
	// SJM CountryCode = 744
	SJM CountryCode = 744
	// SWZ CountryCode = 748
	SWZ CountryCode = 748
	// SWE CountryCode = 752
	SWE CountryCode = 752
	// XSC CountryCode = 826
	XSC CountryCode = 826
	// CHE CountryCode = 756
	CHE CountryCode = 756
	// SYR CountryCode = 760
	SYR CountryCode = 760
	// TWN CountryCode = 158
	TWN CountryCode = 158
	// TJK CountryCode = 762
	TJK CountryCode = 762
	// TZA CountryCode = 834
	TZA CountryCode = 834
	// THA CountryCode = 764
	THA CountryCode = 764
	// TLS CountryCode = 626
	TLS CountryCode = 626
	// TGO CountryCode = 768
	TGO CountryCode = 768
	// TKL CountryCode = 772
	TKL CountryCode = 772
	// TON CountryCode = 776
	TON CountryCode = 776
	// TTO CountryCode = 780
	TTO CountryCode = 780
	// TUN CountryCode = 788
	TUN CountryCode = 788
	// TUR CountryCode = 792
	TUR CountryCode = 792
	// TKM CountryCode = 795
	TKM CountryCode = 795
	// TCA CountryCode = 796
	TCA CountryCode = 796
	// TUV CountryCode = 798
	TUV CountryCode = 798
	// UGA CountryCode = 800
	UGA CountryCode = 800
	// UKR CountryCode = 804
	UKR CountryCode = 804
	// ARE CountryCode = 784
	ARE CountryCode = 784
	// GBR CountryCode = 826
	GBR CountryCode = 826
	// USA CountryCode = 840
	USA CountryCode = 840
	// UMI CountryCode = 581
	UMI CountryCode = 581
	// URY CountryCode = 858
	URY CountryCode = 858
	// XWA CountryCode = 826
	XWA CountryCode = 826
	// UZB CountryCode = 860
	UZB CountryCode = 860
	// VUT CountryCode = 548
	VUT CountryCode = 548
	// VAT CountryCode = 336
	VAT CountryCode = 336
	// VEN CountryCode = 862
	VEN CountryCode = 862
	// VNM CountryCode = 704
	VNM CountryCode = 704
	// VGB CountryCode = 92
	VGB CountryCode = 92
	// VIR CountryCode = 850
	VIR CountryCode = 850
	// WLF CountryCode = 876
	WLF CountryCode = 876
	// ESH CountryCode = 732
	ESH CountryCode = 732
	// YEM CountryCode = 887
	YEM CountryCode = 887
	// YUG CountryCode = 891
	YUG CountryCode = 891
	// ZMB CountryCode = 894
	ZMB CountryCode = 894
	// ZWE CountryCode = 716
	ZWE CountryCode = 716
	// AFG CountryCode = 4
	AFG CountryCode = 4
	// SRB CountryCode = 688
	SRB CountryCode = 688
	// BES CountryCode = 535
	BES CountryCode = 535
	// ALA CountryCode = 248
	ALA CountryCode = 248
	// JEY CountryCode = 832
	JEY CountryCode = 832
	// GGY CountryCode = 831
	GGY CountryCode = 831
	// CUW CountryCode = 531
	CUW CountryCode = 531
	// IMN CountryCode = 833
	IMN CountryCode = 833
	// BLM CountryCode = 652
	BLM CountryCode = 652
	// MAF CountryCode = 663
	MAF CountryCode = 663
	// SXM CountryCode = 534
	SXM CountryCode = 534
	// MNE CountryCode = 499
	MNE CountryCode = 499
	// SSD CountryCode = 728
	SSD CountryCode = 728
	// XKX CountryCode = 900
	XKX CountryCode = 900
	// NON CountryCode = 999
	NON CountryCode = 998
)

// ByName - return CountryCode by country Alpha-2 / Alpha-3 / name, case-insensitive, example: rus := ByName("Ru") OR rus := ByName("russia"),
// returns countries.Unknown, if country name not found or not valid
//
//nolint:misspell,gocyclo
func CountryByName(name string) CountryCode { //nolint:misspell,gocyclo
	switch textPrepare(name) {
	case "AU", "AUS", "AUSTRALIA", "AVSTRALIA", "AVSTRALIYA", "AUSTRALIYA", "AUSTRALIEN":
		return AUS
	case "AT", "AUT", "AUSTRIA", "AVSTRIA", "AUSTRIYA", "AVSTRIYA", "ÖSTERREICH", "OESTERREICH":
		return AUT
	case "AZ", "AZE", "AZERBAIJAN", "AYZERBAIJAN", "AZERBAIDJAN", "AYZERBAIDJAN", "ASERBAIDSCHAN":
		return AZE
	case "AL", "ALB", "ALBANIA", "ALBANIYA", "ALBANIEN":
		return ALB
	case "DZ", "DZA", "ALGERIA", "ALGERIYA", "ALGERIEN":
		return DZA
	case "AS", "ASM", "AMERICANSAMOA", "AMERICASAMOA", "SAMOAAMERICAN", "SAMOAMERICAN", "SAMOAMERICA", "AMERIKANISCHSAMOA":
		return ASM
	case "AI", "AIA", "ANGUILLA", "ANGUILA":
		return AIA
	case "XEN", "ENG", "ENGLAND", "INGLAND":
		return GBR
	case "AO", "AGO", "ANGOLA", "ANGOLIA":
		return AGO
	case "AD", "AND", "ANDORRA", "ANDORA":
		return AND
	case "AQ", "ATA", "NQ", "ATB", "ATN", "BQAQ", "NQAQ", "ANTARCTICA", "ANTARKTICA", "ANTARCTIKA", "ANTARKTIKA", "ANTARCTIC", "ANTARKTIC", "ANTARCTIK", "ANTARKTIK", "ANTARKTIS":
		return ATA
	case "AG", "ATG", "ANTIGUAANDBARBUDA", "ANTIGUABARBUDA", "ANTIGUA", "ANTIGUAUNDBARBUDA":
		return ATG
	case "AN", "ANT", "AHO", "ANHH", "NETHERLANDSANTILLES", "NETHERLSANTILLES", "NETHERLANDSANTILES", "NETHERLSANTILES", "NIEDERLAENDISCHEANTILLEN", "NIEDERLÄNDISCHANTILLEN":
		return ANT
	case "AE", "ARE", "UAE", "UNITEDARABEMIRATES", "ARABEMIRATES", "UNITEDEMIRATES", "VEREINIGTEARABISCHEEMIRATE":
		return ARE
	case "AR", "ARG", "ARGENTINA", "ARGENTIN", "RA", "ARGENTINIEN":
		return ARG
	case "AM", "ARM", "ARMENIA", "ARMENIYA", "ARMENIAN", "ARMENIEN":
		return ARM
	case "AW", "ABW", "ARUBA":
		return ABW
	case "AF", "AFG", "AFGHANISTAN", "AFHANISTAN", "AFGANISTAN", "AFGHANIAN", "AFGANIAN", "AFGHAN", "AFGHANI":
		return AFG
	case "BS", "BHS", "BAHAMAS", "BAGHAMAS", "BAGAMAS", "BAHAMIAN", "BAGAMIAN":
		return BHS
	case "BD", "BGD", "BANGLADESH", "BANGLADEH", "BANHGLADESH", "BANHLADESH", "BANHLADEH":
		return BGD
	case "BB", "BRB", "BAR", "BDS", "BARBADOS", "BARBODOS":
		return BRB
	case "BH", "BHR", "BAHRAIN", "BAGHRAIN":
		return BHR
	case "BY", "BLR", "BYS", "BYAA", "BELARUS", "BELORUS", "BELLARUSSIA", "BELARUSSIA", "BELLORUSSIA", "BELORUSSIA", "BELLARUSSIAN", "BELARUSSIAN", "BELLORUSSIAN", "BELORUSSIAN", "BYELORUSSIAN", "BYELORUSSIA", "BYELORUSSIYA", "WEISSRUSSLAND":
		return BLR
	case "BZ", "BLZ", "BIZ", "BELIZE":
		return BLZ
	case "BE", "BEL", "BELGIUM", "BELGUM", "BELGIEN":
		return BEL
	case "BJ", "BEN", "DHY", "BENIN", "DY", "DYBJ":
		return BEN
	case "BM", "BMU", "BERMUDA", "BERMUDS", "BERMUD":
		return BMU
	case "BG", "BGR", "BULGARIA", "BULGARIYA", "BULGARY", "BOLGARIA", "BOLGARIYA", "BULGARIEN":
		return BGR
	case "BO", "BOL", "BOLIVIA", "BOLIVIYA", "BOLIVIAN", "BOLIVIAPLURINATIONALSTATEOF", "BOLIVIAPLURINATIONALSTATE", "BOLIVIEN":
		return BOL
	case "BA", "BIH", "BOSNIAANDHERZEGOVINA", "BOSNIAHERZEGOVINA", "BOSNIA", "BOSNIEN", "BOSNIENUNDHERZEGOWINA":
		return BIH
	case "BW", "BWA", "BOTSWANA", "BOTSWANNA", "BOTSVANA", "BOTSVANNA":
		return BWA
	case "BR", "BRA", "BRAZIL", "BRAZILIA", "BRAZILIYA", "BRAZILIAN", "BRASILIEN", "REPUBLICOFBRAZIL", "FEDERATIVEREPUBLICOFBRAZIL":
		return BRA
	case "IO", "IOT", "BRITISHINDIANOCEANTERRITORY", "BRITISHINDIANTERRITORY", "BRITISCHESTERRITORIUM", "BRITISCHESTERRITORIUMIMINDISCHENOZEAN":
		return IOT
	case "BN", "BRN", "BRU", "BRUNEI", "BRUNEY", "BRUNEIDARUSSALAM":
		return BRN
	case "BF", "BFA", "HV", "HVO", "BURKINAFASO", "BURKINAANDFASO", "BURCINAFASO", "BURCINAANDFASO", "HVBF":
		return BFA
	case "BI", "BDI", "BURUNDI":
		return BDI
	case "BT", "BTN", "BHUTAN", "BGHUTAN":
		return BTN
	case "VU", "VUT", "NHB", "VANUATU", "NH", "NHVU":
		return VUT
	case "VA", "VAT", "HOLYSEEVATICAN", "HOLYSEE", "VATICAN", "VATICANCITYSTATE", "VATICANSTATE", "HOLYSEEVATIKAN", "VATIKAN", "VATIKANCITYSTATE", "VATIKANSTATE", "HOLYSEEVATIKANCITYSTATE", "VATIKANSTADT", "VATICANCITY", "CITYVATICAN":
		return VAT
	case "GB", "DG", "GBR", "ADN", "DGA", "UNITEDKINGDOM", "UNITEDKINDOM", "UK", "GREATBRITAN", "GREATBRITAIN", "NORTHERNIRELAND", "BRITAN", "BRITAIN", "GROSSBRITANNIEN", "VEREINIGTESKÖNIGREICH", "VEREINIGTESKOENIGREICH": //nolint
		return GBR
	case "HU", "HUN", "HUNGARY", "HUNGAR", "HUNGARI", "VENGRIYA", "VENGRIA", "UNGARN":
		return HUN
	case "VE", "VEN", "VENEZUELA", "VENEZUELLA", "VENECUELA", "VENECUELLA", "YV", "BOLIVARIANREPUBLICOF", "BOLIVARIANREPUBLIC", "REPUBLICOFBOLIVARIAN", "REPUBLICBOLIVARIAN", "BOLIVARIAN": //nolint
		return VEN
	case "VG", "VGB", "IVB", "VIRGINISLANDSBRITISH", "VIRGINISLANDSBRITIH", "VIRGINISLSBRITIH", "VIRGINISLSBRITISH", "VIRGINISLANDSGB", "VIRGINISLANDSUK", "BRITISCHEJUNGFERNINSELN", "BRITISHVIRGINISLANDS":
		return VGB
	case "VI", "VIR", "ISV", "VIRGINISLANDSUS", "USVIRGINISLANDS", "USVI", "AMERIKANISCHEJUNGFERNINSELN":
		return VIR
	case "TL", "TP", "TLS", "TMP", "TPTL", "TIMORLESTE", "EASTTIMOR", "TIMOR", "TIMORELESTE", "EASTTIMORE", "TIMORE", "TIMORLESTEEASTTIMORE", "OSTTIMOR":
		return TLS
	case "VN", "VNM", "VIE", "VDR", "VD", "VIETNAM", "VETNAM", "VIETNAME", "VETNAME", "VDVN", "VIỆTNAM", "CỘNGHÒAXÃHỘICHỦNGHĨAVIỆTNAM", "CHỦNGHĨAVIỆTNAM", "NGHĨAVIỆTNAM":
		return VNM
	case "GA", "GAB", "GABON", "GABUN":
		return GAB
	case "HT", "HTI", "HAITI", "GAITI":
		return HTI
	case "GY", "GUY", "GUYANA":
		return GUY
	case "GM", "GMB", "WAG", "GAMBIA", "GAMBIYA":
		return GMB
	case "GH", "GHA", "GHANA", "HANA":
		return GHA
	case "GP", "GLP", "GUADELOUPE", "GUADELUPE", "GUADELOOPE", "GUADELOUPA", "GUADELUPA", "GUADELOOPA":
		return GLP
	case "GT", "GTM", "GCA", "GUATEMALA":
		return GTM
	case "GN", "GIN", "GUINEA", "GUINEYA":
		return GIN
	case "GW", "GNB", "GBS", "GUINEABISSAU":
		return GNB
	case "DE", "DEU", "DD", "DDR", "GER", "GERMANY", "GERMANIYA", "DEUTSCHLAND", "DEUTSCH", "DDDE":
		return DEU
	case "GI", "GIB", "GBZ", "GIBRALTAR", "HIBRALTAR":
		return GIB
	case "HN", "HND", "HONDURAS", "GONDURAS":
		return HND
	case "HK", "HKG", "HONGKONG", "HONKONG":
		return HKG
	case "GD", "GRD", "GRENADA", "GRINADA", "WG":
		return GRD
	case "GL", "GRL", "GREENLAND", "GRÖNLAND", "GROENLAND":
		return GRL
	case "GR", "GRC", "GREECE", "GRECE", "GRIECHENLAND", "GRECIYA":
		return GRC
	case "GE", "GEO", "GEORGIA", "GEORGIYA", "GEORGIEN", "GRUZIYA":
		return GEO
	case "GU", "GUM", "GUAM":
		return GUM
	case "DK", "DNK", "DENMARK", "DANMARK", "DÄNEMARK", "DAENEMARK", "KONGERIGETDANMARK", "DANMARKKONGERIGET", "DANIYA":
		return DNK
	case "CD", "COD", "ZRE", "ZAR", "ZR", "ZRCD", "CONGODEMOCRATICREPUBLIC", "DEMOCRATICREPUBLICOFTHECONGO", "CONGODEMOCRATICREP", "CONGODEMOCRATIC", "CONGOTHEDEMOCRATICREPUBLICOF", "CONGOTHEDEMOCRATICREPUBLIC", "KONGODEMOCRACTICREPUBLIC", "KONGODEMOCRATICREP", "KONGODEMOCRATIC", "KONGOTHEDEMOCRATICREPUBLICOF", "ZAIRE", "ZAIR", "DEMOKRATISCHEREPUBLIKKONGO", "CONGOREPUBLIC", "KONGOREPUBLIC", "REPUBLICOFCONGO", "REPUBLICOFKONGO", "CONGOTHEDEMOCRATICREPUBLICOFTHE", "DRCONGO":
		return COD
	case "DJ", "DJI", "AFI", "DJIBOUTI", "AIDJ", "DSCHIBUTI":
		return DJI
	case "DM", "DMA", "DOMINICA", "DOMINIKA":
		return DMA
	case "DO", "DOM", "DOMINICANREPUBLIC", "DOMINICANA", "DOMINIKANA", "DOMINIKANISCHEREPUBLIK":
		return DOM
	case "EG", "EGY", "EGYPT", "ÄGYPTEN", "AEGYPTEN":
		return EGY
	case "ZM", "ZMB", "RNR", "ZAMBIA", "SAMBIA":
		return ZMB
	case "EH", "ESH", "WESTERNSAHARA", "WESTSAHARA":
		return ESH
	case "ZW", "ZWE", "ZIM", "RHO", "RSR", "ZIMBABWE", "ZIMBABVE", "RH", "RHZW", "SIMBABWE":
		return ZWE
	case "IL", "ISR", "ISRAEL", "IZRAIL", "ISRAIL", "ISRAILIAN", "IZRAILEN":
		return ISR
	case "IN", "IND", "INDIA", "INDIAN", "INDIYA", "SKM", "SKIN", "INDIEN":
		return IND
	case "ID", "IDN", "INA", "INDONESIA", "REPUBLICOFINDONESIA", "RI", "INDONESIEN":
		return IDN
	case "JO", "JOR", "HKJ", "JORDAN", "JORDANIEN":
		return JOR
	case "IQ", "IRQ", "IRAQ", "IRAK":
		return IRQ
	case "IR", "IRN", "IRI", "IRAN", "IRANISLAMICREPUBLICOF", "IRANISLAMICREPUBLIC", "IRANIAN":
		return IRN
	case "IE", "IRL", "IRELAND", "IRLAND":
		return IRL
	case "IS", "ISL", "ICELAND", "ISLAND":
		return ISL
	case "ES", "EA", "IC", "ESP", "SPAIN", "SPANIEN", "ISPANIA":
		return ESP
	case "IT", "ITA", "ITALY", "ITALIYA", "ITALIEN":
		return ITA
	case "YE", "YEM", "YMD", "YEMEN", "IEMEN", "YD", "YDYE", "JEMEN":
		return YEM
	case "KZ", "KAZ", "KAZAKHSTAN", "KAZAHSTAN", "KASACHSTAN":
		return KAZ
	case "KY", "CYM", "CAYMANISLANDS", "KAYMANISLANDS", "KAIMANINSELN":
		return CYM
	case "KH", "KHM", "CAMBODIA", "KAMBODSCHA":
		return KHM
	case "CM", "CMR", "CAMEROON", "KAMERUN":
		return CMR
	case "CA", "CAN", "CDN", "CANADA", "KANADA":
		return CAN
	case "QA", "QAT", "QATAR", "KATAR":
		return QAT
	case "KE", "KEN", "EAK", "KENYA":
		return KEN
	case "CY", "CYP", "CYPRUS", "CIPRUS", "ZYPERN", "REPUBLIKZYPERN":
		return CYP
	case "KI", "KIR", "CT", "CTE", "CTKI", "KIRIBATI", "CIRIBATI", "KIRIBATY", "CIRIBATY":
		return KIR
	case "CN", "CHN", "CHINA", "CHINESE", "RC", "KITAY":
		return CHN
	case "CC", "CCK", "KEELING", "COCOS", "COCOSKEELINGISLANDS", "COCOSISLANDS", "KOKOSISLANDS", "KOKOSINSELN":
		return CCK
	case "CO", "COL", "COLOMBIA", "KOLUMBIEN":
		return COL
	case "KM", "COM", "COMOROS", "KOMOREN":
		return COM
	case "CG", "COG", "RCB", "CONGO", "KONGO":
		return COG
	case "KP", "PRK", "DEMOCRATICPEOPLESREPUBLICOFKOREA", "KOREADEMOCRATICPEOPLESREPUBLICOF", "KOREADEMOCRATICPEOPLESREPUBLIC", "KOREANORTH", "NORTHKOREA", "NORDKOREA":
		return PRK
	case "KR", "KOR", "ROK", "KOREA", "KOREYA", "SOUTHKOREA", "KOREAREPUBLICOF", "KOREAREPUBLIC", "REPUBLICOFKOREA", "KOREAREPOF", "SÜDKOREA", "SUEDKOREA":
		return KOR
	case "CR", "CRI", "COSTARICA", "KOSTARIKA", "KOSTARICA", "COSTARIKA":
		return CRI
	case "CI", "CIV", "COTEDIVOIRE", "CÔTEDIVOIRE", "IVORYCOAST", "ELFENBEINKÜSTE", "ELFENBEINKUESTE":
		return CIV
	case "CU", "CUB", "CUBA", "CUBAREPUBLIC", "REPUBLICCUBA", "KUBA":
		return CUB
	case "KW", "KWT", "KUWAIT":
		return KWT
	case "KG", "KGZ", "KYRGYZSTAN", "KIRGISISTAN":
		return KGZ
	case "LA", "LAO", "LAOS", "LAODEMOCRATICPEOPLESREPUBLIC", "LAOSDEMOCRATICPEOPLESREPUBLIC", "LAOPEOPLESDEMOCRATICREPUBLIC":
		return LAO
	case "LV", "LVA", "LAT", "LATVIA", "LATVIYA", "LETTLAND":
		return LVA
	case "LS", "LSO", "LESOTHO":
		return LSO
	case "LR", "LBR", "LIBERIA":
		return LBR
	case "LB", "LBN", "LEBANON", "RL", "LIBANON":
		return LBN
	case "LY", "LBY", "LBA", "LIBYA", "LIVIA", "LIVIYA", "LIBYAN", "LIBYANARABJAMAHIRIYA", "LF", "LIBYEN":
		return LBY
	case "LT", "LTU", "LITHUANIA", "LITAUEN", "LITVA":
		return LTU
	case "LI", "LIE", "LIECHTENSTEIN", "LIEHTENSTEIN", "FL":
		return LIE
	case "LU", "LUX", "LUXEMBOURG", "LUXEMBURG":
		return LUX
	case "MU", "MUS", "MAURITIUS":
		return MUS
	case "MR", "MRT", "MAURITANIA", "MAURETANIEN":
		return MRT
	case "MG", "MDG", "MADAGASCAR", "RM", "MADAGASKAR":
		return MDG
	case "YT", "MYT", "MAYOTTE":
		return MYT
	case "MO", "MAC", "MACAUCHINA", "MACAU", "MACAO", "MACAUSAR", "MACAOSAR":
		return MAC
	case "MK", "MKD", "MACEDONIA", "MACEDONIAFYRO", "MACEDONIATHEFORMERYUGOSLAVREPUBLICOF", "MACEDONIATHEFORMERYUGOSLAV", "MACEDONIATHEFORMERYUGOSLAVREPUBLIC", "REPUBLICOFNORTHMACEDONIA", "REPUBLICOFMACEDONIA", "NORTHMACEDONIA", "MACEDONIANORTH", "NORDMAZEDONIEN", "THEFORMERYUGOSLAVREPUBLICOF", "THEFORMERYUGOSLAVREPUBLIC", "FORMERYUGOSLAVREPUBLICOF", "FORMERYUGOSLAVREPUBLIC", "MACEDONIAFORMERYUGOSLAVREPUBLICOF", "MACEDONIAFORMERYUGOSLAVREPUBLIC", "YUGOSLAVREPUBLIC":
		return MKD
	case "MW", "MWI", "MAW", "MALAWI", "MALAVI":
		return MWI
	case "MY", "MYS", "MAL", "MALAYSIA", "MALAYSIYA":
		return MYS
	case "ML", "MLI", "RMM", "MALI":
		return MLI
	case "MV", "MDV", "MALDIVES", "MALEDIVEN":
		return MDV
	case "MT", "MLT", "MALTA":
		return MLT
	case "MP", "MNP", "NORTHERNMARIANAISLANDS", "NORTHERNMARIANAIS", "MARIANAISLANDS", "NÖRDLICHEMARIANEN", "NOERDLICHEMARIANEN":
		return MNP
	case "MA", "MAR", "MOROCCO", "MOROCO", "MOROKO", "MAROKKO":
		return MAR
	case "MQ", "MTQ", "MARTINIQUE":
		return MTQ
	case "MH", "MHL", "MARSHALLISLANDS", "MARSHALL", "REPUBLICOFTHEMARSHALLISLANDS", "MARSHALLINSELN":
		return MHL
	case "MX", "MEX", "MEXICO", "MEXIKO":
		return MEX
	case "FM", "FSM", "MICRONESIA", "MICRONESIAFEDERATEDSTATESOF", "MICRONESIAFEDST", "MIKRONESIEN", "FEDERATEDSTATESOFMICRONESIA", "STATESOFMICRONESIA", "FEDERATEDSTATESMICRONESIA", "STATESMICRONESIA":
		return FSM
	case "MZ", "MOZ", "MOZAMBIQUE", "MOZAMBIQ", "MOSAMBIK":
		return MOZ
	case "MD", "MDA", "MOLDOVA", "MOLDAVIA", "MOLDAVIAN", "MOLDAVIYA", "REPUBLIKMOLDOVA", "REPUBLICOFMOLDOVA", "MOLDOVAREPUBLICOF", "MOLDOVAREPUBLIC":
		return MDA
	case "MC", "MCO", "MONACO", "MONAKO":
		return MCO
	case "MN", "MNG", "MONGOLIA", "MONGOLIAN", "MONGOLIYA", "MONGOLEI":
		return MNG
	case "MS", "MSR", "MONTSERRAT":
		return MSR
	case "MM", "BU", "MMR", "BUMM", "MYANMAR", "BURMA":
		return MMR
	case "NA", "NAM", "NAMIBIA", "NAMIBIAN", "NAMIBIYA", "NAMIBIE":
		return NAM
	case "NR", "NRU", "NAURU":
		return NRU
	case "NP", "NPL", "NEPAL", "NEPALI":
		return NPL
	case "NE", "NER", "NIGER", "NIGGER", "RN":
		return NER
	case "NG", "NGA", "NGR", "WAN", "NIGERIA", "NIGERIAN", "NIGGERIAN", "NIGERIYA", "NIGGERIA", "NIGGERIYA":
		return NGA
	case "NL", "NLD", "NED", "NETHERLANDS", "NETHERLAND", "HOLLAND", "HOLLANDIA", "HOLLANDIYA", "NIEDERLANDE", "HOLAND", "HOLANDIA", "HOLANDIYA":
		return NLD
	case "NI", "NIC", "NICARAGUA":
		return NIC
	case "NU", "NIU", "NIUE":
		return NIU
	case "NZ", "NZL", "NEWZEALAND", "NEWZELANDIA", "NEWZELAND", "NEUSEELAND":
		return NZL
	case "NC", "NCL", "NEWCALEDONIA", "NEWCALEDONIYA", "NEUKALEDONIEN":
		return NCL
	case "NO", "NOR", "NORWAY", "NORWEGEN":
		return NOR
	case "OM", "OMN", "OMAN":
		return OMN
	case "BV", "BVT", "BOUVET", "BOUVETE", "BOUVETISLAND", "ISLANDOFBOUVET", "BOUVETINSEL":
		return BVT
	case "IM", "IMN", "GBM", "ISLEOFMAN":
		return IMN
	case "NF", "NFK", "NORFOLKISLAND", "NORFOLK", "NORFOLCISLAND", "NORFOLC", "NORFOLKINSEL":
		return NFK
	case "PN", "PCN", "PITCAIRN", "THEPITCAIRN", "PITCAIRNISLANDS", "THEPITCAIRNISLANDS", "DUCIEANDOENOISLANDS", "DUCIEANDOENO", "PITCAIRNINSELN":
		return PCN
	case "CX", "CXR", "CHRISTMASISLAND", "TERRITORYOFCHRISTMASISLAND", "WEIHNACHTSINSEL":
		return CXR
	case "SH", "TA", "SHN", "TAA", "ASC", "SAINTHELENA", "SAINTELENA", "STHELENA", "STELENA", "TRISTAN", "ASCENSIONANDTRISTANDACUNHA", "ASCENSIONTRISTANDACUNHA", "TRISTANDACUNHA", "SANKTHELENA":
		return SHN
	case "WF", "WLF", "WALLISANDFUTUNAISLANDS", "WALLISFUTUNAISLANDS", "WALLISANDFUTUNA", "WALLISFUTUNA", "WALLISUNDFUTUNA":
		return WLF
	case "HM", "HMD", "HEARDISLANDANDMCDONALDISLANDS", "HEARDISLAND", "HEARDUNDMCDONALDINSELN", "HEARDANDMCDONALDISLANDS":
		return HMD
	case "CV", "CPV", "CAPEVERDE", "KAPVERDE", "CABOVERDE":
		return CPV
	case "CK", "COK", "COOKISLANDS", "COOKINSELN":
		return COK
	case "WS", "WSM", "SAMOA":
		return WSM
	case "SJ", "SJM", "SVALBARDANDJANMAYENISLANDS", "SVALBARD", "SVALBARDUNDJANMAYEN", "SVALBARDANDJANMAYEN":
		return SJM
	case "TC", "TCA", "TURKSANDCAICOSISLANDS", "TURKSANDCAICOSIS", "CAICOSISLANDS", "CACOSISLANDS", "TURKSUNDCACIOINSELN":
		return TCA
	case "UM", "UMI", "UNITEDSTATESMINOROUTLYINGISLANDS", "MINOROUTLYINGISLANDS", "MINOROUTLYING", "USMI", "JT", "JTN", "JTUM", "MI", "MID", "MIUM", "PU", "PUS", "PUUM", "WK", "WAK", "WKUM", "KLEINEINSELBESITZUNGENDERVEREINIGTENSTAATEN", "USOUTLYINGISLANDS":
		return UMI
	case "PK", "PAK", "PAKISTAN", "PACISTAN":
		return PAK
	case "PW", "PLW", "PALAU":
		return PLW
	case "PS", "PSE", "PLE", "PALESTINE", "PALESTINA", "PALESTINIAN", "PALESTINIANTERRITORY", "PALÄSTINA", "PALAESTINA", "OCCUPIEDPALESTINIANTERRITORY":
		return PSE
	case "PA", "PAN", "PCZ", "PANAMA", "PANAMIAN", "PANAM", "PZ", "PZPA":
		return PAN
	case "PG", "PNG", "PAPUANEWGUINEA", "PAPUA", "PAPUANEUGUINEA", "NEWGUINEA", "NEUGUINEA":
		return PNG
	case "PY", "PRY", "PARAGUAY":
		return PRY
	case "PE", "PER", "PERU":
		return PER
	case "PL", "POL", "POLAND", "POLSKI", "POLSHA", "POLEN":
		return POL
	case "PT", "PRT", "PORTUGAL", "PORTUGALIAN", "PORTUGALIYA":
		return PRT
	case "PR", "PRI", "PUERTORICO", "PUERTORIKO":
		return PRI
	case "RE", "REU", "REUNION", "RÉUNION":
		return REU
	case "RU", "RUS", "SUN", "RUSSIA", "RUSSO", "RUSSISH", "RUSSLAND", "RUSLAND", "RUSIA", "ROSSIA", "ROSSIYA", "RUSSIAN", "RUSSIANFEDERATION", "USSR":
		return RUS
	case "RW", "RWA", "RWANDA", "RUANDA", "RUWANDA":
		return RWA
	case "RO", "ROU", "ROM", "ROMANIA", "RUMINIA", "RUMINIYA", "RUMÄNIEN", "RUMAENIEN":
		return ROU
	case "SV", "SLV", "ESA", "ELSALVADOR":
		return SLV
	case "SM", "SMR", "RSM", "SANMARINO":
		return SMR
	case "ST", "STP", "SAOTOMEANDPRINCIPE", "SAOTOME", "SAOTOMEUNDPRINCIPE", "SÃOTOMÉANDPRÍNCIPE":
		return STP
	case "SA", "SAU", "SAUDIARABIA", "SAUDI", "SAUDIARABIEN":
		return SAU
	case "SZ", "SWZ", "SWAZILAND", "SWASILAND", "ESWATINI", "KINGDOMOFESWATINI", "KINGDOMESWATINI", "SVAZILEND":
		return SWZ
	case "SC", "SYC", "SEYCHELLES", "SEYCHELLEN":
		return SYC
	case "SN", "SEN", "SENEGAL":
		return SEN
	case "PM", "SPM", "SAINTPIERREANDMIQUELON", "SAINTPIERRE", "STPIERREANDMIQUELON", "STPIERRE", "SANKTPIERRE", "SANKTPIERREUNDMIQUELON":
		return SPM
	case "VC", "VCT", "SAINTVINCENTANDTHEGRENADINES", "SAINTVINCENT", "STVINCENTANDTHEGRENADINES", "STVINCENT", "WV", "STVINCENTUNDDIEGRENADINEN", "STVINCENTANDGRENADINES":
		return VCT
	case "KN", "KNA", "SAINTKITTSANDNEVIS", "SAINTKITTSNEVIS", "SAINTKITTS", "STKITTSANDNEVIS", "STKITTSNEVIS", "STKITTS", "SANKTKITTSUNDNEVIS":
		return KNA
	case "LC", "LCA", "SAINTLUCIA", "STLUCIA", "LUCIA", "WL":
		return LCA
	case "SG", "SGP", "SINGAPORE", "SINGPAORE", "SINGAPORECITY", "SINGAPOUR", "SINGAPURA", "SINGAPUR": //nolint
		return SGP
	case "SY", "SYR", "SYRIA", "SYRIAN", "SYRIANARABREPUBLIC", "SYRIEN":
		return SYR
	case "SK", "SVK", "CSHH", "SLOVAKIA", "SLOVAK", "SLOVAKIYA", "SLOVACIA", "SLOVAC", "SLOVACIYA", "SLOWAKEI":
		return SVK
	case "SI", "SVN", "SLO", "SLOVENIA", "SLOVENIYA", "SLOWENIEN":
		return SVN
	case "US", "USA", "UNITEDSTATES", "UNITEDSTATESOFAMERICA", "USOFAMERICA", "USAMERICA", "VEREINIGTESTAATENVONAMERIKA":
		return USA
	case "SB", "SLB", "SOLOMONISLANDS", "SOLOMON", "SALOMONEN":
		return SLB
	case "SO", "SOM", "SOMALIA", "SOMALI":
		return SOM
	case "SD", "SDN", "SUDAN", "SUDANE", "UMHŪRIYYATASSŪDĀN", "SŪDĀN", "جمهوريةالسودان", "السودان":
		return SDN
	case "SR", "SUR", "SME", "SURINAME", "SURINAM":
		return SUR
	case "SL", "SLE", "WAL", "SIERRALEONE", "SIERRALEON", "SIERALEONE", "SIERALEON":
		return SLE
	case "TJ", "TJK", "TAJIKISTAN", "TADJIKISTAN", "TADSCHIKISTAN":
		return TJK
	case "TW", "TWN", "TPE", "TAIWAN", "TAIWANIAN", "PROVINCEOFCHINA", "PROVINCECHINA":
		return TWN
	case "TH", "THA", "THAILAND", "TAILAND", "THAI", "THAYLAND", "TAYLAND":
		return THA
	case "TZ", "TZA", "EAT", "EAZ", "TANZANIA", "TANZANIYA", "TANSANIA", "TANZANIAUNITEDREPUBLICOF", "TANZANIAUNITEDREPUBLIC", "REPUBLICOFTANZANIA", "TANZANIAREPUBLIC":
		return TZA
	case "TG", "TGO", "TOGO":
		return TGO
	case "TK", "TKL", "TOKELAU":
		return TKL
	case "TO", "TON", "TONGA":
		return TON
	case "TT", "TTO", "TRINIDADANDTOBAGO", "TRINIDAD", "TRINADUNDTOBAGO":
		return TTO
	case "TV", "TUV", "TUVALU":
		return TUV
	case "TN", "TUN", "TUNISIA", "TUNESIEN":
		return TUN
	case "TM", "TKM", "TMN", "TURKMENISTAN", "TURKMENISTON", "TURKMENI", "TURKMENIA", "TURKMENIYA":
		return TKM
	case "TR", "TUR", "TURKEY", "TURCIA", "TURKISH", "TÜRKEI", "TUERKEI", "TÜRKIYE", "REPUBLICOFTÜRKIYE", "TÜRKIYEREPUBLICOF", "TÜRKIYEREPUBLIC", "REPUBLICTÜRKIYE":
		return TUR
	case "UG", "UGA", "EAU", "UGANDA":
		return UGA
	case "UZ", "UZB", "UZBEKISTAN", "UZBEKISTON":
		return UZB
	case "UA", "UKR", "UKRAINE", "UKRAINA": //nolint
		return UKR
	case "UY", "URY", "URUGUAY", "URUGWAY":
		return URY
	case "XW", "XWA", "WALES":
		return XWA
	case "FO", "FRO", "FAROEISLANDS", "FAROE", "FÄRÖER", "FAEROERER":
		return FRO
	case "FJ", "FJI", "FIJI", "FIDSCHI":
		return FJI
	case "PH", "PHL", "PHI", "PHILIPPINES", "PHILIPINES", "PI", "RP", "PHILIPPINEN": //nolint
		return PHL
	case "FI", "SF", "FIN", "FINLAND", "FINNISH", "FINNLAND":
		return FIN
	case "FK", "FLK", "FALKLANDISLANDSMALVINAS", "MALVINAS", "FALKLANDISLANDS", "FALKLAND", "FALKLANDINSELN":
		return FLK
	case "FR", "CP", "FX", "FRA", "FXX", "CPT", "FXFR", "FRANCE", "FRENCH", "FRANKREICH":
		return FRA
	case "GF", "GUF", "FRENCHGUIANA", "GUIANA", "FRANZÖSISCHGUYANA", "FRANZOESISCHGUYANA":
		return GUF
	case "PF", "PYF", "FRENCHPOLYNESIA", "POLYNESIA", "FRANZÖSISCHPOLYNESIEN", "FRANZOESISCHPOLYNESIEN":
		return PYF
	case "TF", "ATF", "FRENCHSOUTHERNTERRITORIES", "SOUTHERNTERRITORIESFRENCH", "FRANZÖSISCHESÜDUNDANTARKTISGEBIETE", "FRANZOESISCHESUEDUNDANTARKTISGEBIETE":
		return ATF
	case "HR", "HRV", "CRO", "CROATIA", "KROATIA", "KROATIEN":
		return HRV
	case "CF", "CAF", "CTA", "RCA", "CENTRALAFRICANREPUBLIC", "CENTRALAFRICANREP", "CENTRALAFRICAN", "ZENTRALAFRIKA":
		return CAF
	case "TD", "TCD", "CHAD", "TSCHAD":
		return TCD
	case "CZ", "CZE", "CZECHIA", "CZECHIYA", "CZECHREPUBLIC", "REPUBLICOFCZECH", "CZECH", "TSCHECHIEN", "CHEHIA", "CHEHIYA":
		return CZE
	case "CL", "CHL", "RCH", "CHILE", "CHILI", "CHILLE":
		return CHL
	case "CH", "CHE", "SWITZERLAND", "SWISS", "SCHWEIZ", "SUISSE", "SVIZZERA", "SVIZRA", "HELVETIA", "SHVEYCARIA", "SHVEYCARIYA":
		return CHE
	case "SE", "SWE", "SWEDEN", "SCHWEDEN", "SHWEDEN", "SHVECIA", "SHVECIYA":
		return SWE
	case "XS", "XSC", "SCOTLAND", "SCHOTTLAND":
		return XSC
	case "LK", "LKA", "SRILANKA":
		return LKA
	case "EC", "ECU", "ECUADOR":
		return ECU
	case "GQ", "GNQ", "EQG", "GEQ", "EQUATORIALGUINEA", "ÄQUATORIALGUINEA", "AEQUATORIALGUINEA":
		return GNQ
	case "ER", "ERI", "ERITREA":
		return ERI
	case "EE", "EST", "ESTONIA", "EW", "ESTLAND":
		return EST
	case "ET", "ETH", "ETHIOPIA", "ÄTHOPIEN", "AETHOPIEN":
		return ETH
	case "ZA", "ZAF", "SOUTHAFRICA", "SÜDAFRIKA", "SUEDAFRIKA":
		return ZAF
	case "YU", "YUG", "YUGOSLAVIA", "UGOSLAVIA", "YUGOSLAVIYA", "UGOSLAVIYA", "SERBIAANDMONTENEGRO", "CS", "SCG", "JUGOSLAWIEN":
		return YUG
	case "GS", "SGS", "SOUTHGEORGIAANDTHESOUTHSANDWICHISLANDS", "SOUTHGEORGIAANDTHESOUTHSANDWICH", "SOUTHGEORGIATHESOUTHSWICHISLANDS", "SOUTHGEORGIA", "SÜDGEORGIEN", "SUEDGEORGIEN":
		return SGS
	case "JM", "JAM", "JAMAICA", "JAMAIKA", "YAMAICA", "YAMAIKA", "JA":
		return JAM
	case "ME", "MNE", "MONTENEGRO":
		return MNE
	case "BL", "BLM", "SAINTBARTHELEMY", "STBARTHELEMY", "SAINTBARTHÉLEMY", "STBARTHÉLEMY":
		return BLM
	case "SX", "SXM", "SINTMAARTENDUTCH", "SAINTMAARTEN", "SINTMAARTEN", "STMAARTEN":
		return SXM
	case "RS", "SRB", "CSXX", "SERBIA", "SERBIYA", "SERBIEN":
		return SRB
	case "AX", "ALA", "ALANDISLANDS", "ISLANDSALAND", "ALAND", "ÅLANDISLANDS", "ÅLAND", "ISLANDSÅLAND":
		return ALA
	case "BQ", "BES", "BONAIRE", "BONAIR", "BONEIRU", "BONAIRESINTEUSTATIUSANDSABA", "BONAIRESINTEUSTATIUSSABA", "BONAIRESTEUSTANDSABA", "BONAIRESTEUSTSABA", "SINTEUSTATIUSANDSABA", "SINTEUSTATIUS", "CARIBBEANNETHERLANDS":
		return BES
	case "GG", "GGY", "GBA", "GBG", "GUERNSEY":
		return GGY
	case "JE", "JEY", "GBJ", "JERSEY", "JERSIEY":
		return JEY
	case "CW", "CUW", "CURACAO", "CURAÇAO", "CURAQAO", "CURAKAO", "KURACAO", "KURAKAO":
		return CUW
	case "MF", "MAF", "SAINTMARTINFRENCH", "STMARTINFRENCH", "SANKTMARTIN", "SAINTMARTIN":
		return MAF
	case "SS", "SSD", "SOUTHSUDAN", "SOUTHSUDANE", "REPUBLICOFSOUTHSUDAN", "SOUTHSUDANREPUBLICOF", "SOUTHSUDANREPUBLIC", "PAGUOTTHUDÄN", "SÜDSUDAN", "SUEDSUDAN":
		return SSD
	case "JP", "JPN", "JAPAN":
		return JPN
	case "XK", "XKX", "XKS", "KOS", "KOSOVO", "COSOVO", "КОСОВО", "KOSOVËS", "РЕПУБЛИКАКОСОВО", "REPUBLIKAKOSOVO", "REPUBLIKACOSOVO", "REPUBLIKAKOSOVËS", "REPUBLICAKOSOVO", "REPUBLICACOSOVO", "REPUBLICAKOSOVËS", "KOSOVOREPUBLIC", "COSOVOREPUBLIC", "KOSOVËSREPUBLIC":
		return XKX
	case "XX", "NONE", "NON", "NICHT", "NICHTS":
		return None
	case "INTERNATIONAL":
		return International
	case "UIFN", "INTERNATIONALFREEPHONE", "TOLLFREEPHONE":
		return NonCountryInternationalFreephone
	case "INMARSAT":
		return NonCountryInmarsat
	case "MMS", "MARITIMEMOBILESERVICE", "MARITIMEMOBILESERVICES", "MARITIMEMOBILE", "MARITIME":
		return NonCountryMaritimeMobileService
	case "UNIVERSALPERSONALTELECOMMUNICATIONSSERVICES", "UNIVERSALPERSONALTELECOMMUNICATIONSSERVICE", "UNIVERSALPERSONALTELECOMMUNICATIONS", "UNIVERSALPERSONALTELECOMMUNICATION":
		return NonCountryUniversalPersonalTelecommunicationsServices
	case "NCP", "NATIONALNONCOMMERCIALPURPOSES", "NONCOMMERCIALPURPOSES", "NATIONALNONCOMMERCIAL", "NONCOMMERCIAL":
		return NonCountryNationalNonCommercialPurposes
	case "GMSS", "GLOBALMOBILESATELLITESYSTEM", "GLOBALMOBILESATELITESYSTEM", "GLOBALMOBILESATELLITE", "GLOBALMOBILESATELITE":
		return NonCountryGlobalMobileSatelliteSystem
	case "INTERNATIONALNETWORKS", "INTERNATIONALNETWORKSSERVICE", "INTERNATIONALNETWORKSSERVICES":
		return NonCountryInternationalNetworks
	case "DISASTERRELIEF", "DISASTER":
		return NonCountryDisasterRelief
	case "IPRS", "INTERNATIONALPREMIUMRATESERVICE", "PREMIUMRATESERVICE", "INTERNATIONALPREMIUMRATESERVICES", "PREMIUMRATESERVICES":
		return NonCountryInternationalPremiumRateService
	case "ITPCS", "INTERNATIONALTELECOMMUNICATIONSPUBLICCORRESPONDENCESERVICETRIAL", "INTERNATIONALTELECOMMUNICATIONSPUBLICCORRESPONDENCESERVICE", "InternationalTELECOMMUNICATIONSPUBLICCORRESPONDENCESERVICES", "InternationalTELECOMMUNICATIONSCORRESPONDENCESERVICE", "InternationalTELECOMMUNICATIONSCORRESPONDENCESERVICES":
		return NonCountryInternationalTelecommunicationsCorrespondenceService
	}
	return Unknown
}

// Alpha3 - returns a Alpha-3 (ISO3, 3 chars) code of country
//
//nolint:gocyclo
func (c CountryCode) Alpha3() string { //nolint:gocyclo
	switch c {
	case 8:
		return "ALB"
	case 12:
		return "DZA"
	case 16:
		return "ASM"
	case 20:
		return "AND"
	case 24:
		return "AGO"
	case 660:
		return "AIA"
	case 10:
		return "ATA"
	case 28:
		return "ATG"
	case 32:
		return "ARG"
	case 51:
		return "ARM"
	case 533:
		return "ABW"
	case 36:
		return "AUS"
	case 40:
		return "AUT"
	case 31:
		return "AZE"
	case 44:
		return "BHS"
	case 48:
		return "BHR"
	case 50:
		return "BGD"
	case 52:
		return "BRB"
	case 112:
		return "BLR"
	case 56:
		return "BEL"
	case 84:
		return "BLZ"
	case 204:
		return "BEN"
	case 60:
		return "BMU"
	case 64:
		return "BTN"
	case 68:
		return "BOL"
	case 70:
		return "BIH"
	case 72:
		return "BWA"
	case 74:
		return "BVT"
	case 76:
		return "BRA"
	case 86:
		return "IOT"
	case 96:
		return "BRN"
	case 100:
		return "BGR"
	case 854:
		return "BFA"
	case 108:
		return "BDI"
	case 116:
		return "KHM"
	case 120:
		return "CMR"
	case 124:
		return "CAN"
	case 132:
		return "CPV"
	case 136:
		return "CYM"
	case 140:
		return "CAF"
	case 148:
		return "TCD"
	case 152:
		return "CHL"
	case 156:
		return "CHN"
	case 162:
		return "CXR"
	case 166:
		return "CCK"
	case 170:
		return "COL"
	case 174:
		return "COM"
	case 178:
		return "COG"
	case 180:
		return "COD"
	case 184:
		return "COK"
	case 188:
		return "CRI"
	case 384:
		return "CIV"
	case 191:
		return "HRV"
	case 192:
		return "CUB"
	case 196:
		return "CYP"
	case 203:
		return "CZE"
	case 208:
		return "DNK"
	case 262:
		return "DJI"
	case 212:
		return "DMA"
	case 214:
		return "DOM"
	case 218:
		return "ECU"
	case 818:
		return "EGY"
	case 222:
		return "SLV"
	case 226:
		return "GNQ"
	case 232:
		return "ERI"
	case 233:
		return "EST"
	case 231:
		return "ETH"
	case 238:
		return "FLK"
	case 242:
		return "FJI"
	case 246:
		return "FIN"
	case 250:
		return "FRA"
	case 234:
		return "FRO"
	case 254:
		return "GUF"
	case 258:
		return "PYF"
	case 260:
		return "ATF"
	case 266:
		return "GAB"
	case 270:
		return "GMB"
	case 268:
		return "GEO"
	case 276:
		return "DEU"
	case 288:
		return "GHA"
	case 292:
		return "GIB"
	case 300:
		return "GRC"
	case 304:
		return "GRL"
	case 308:
		return "GRD"
	case 312:
		return "GLP"
	case 316:
		return "GUM"
	case 320:
		return "GTM"
	case 324:
		return "GIN"
	case 624:
		return "GNB"
	case 328:
		return "GUY"
	case 332:
		return "HTI"
	case 334:
		return "HMD"
	case 340:
		return "HND"
	case 344:
		return "HKG"
	case 348:
		return "HUN"
	case 352:
		return "ISL"
	case 356:
		return "IND"
	case 360:
		return "IDN"
	case 364:
		return "IRN"
	case 368:
		return "IRQ"
	case 372:
		return "IRL"
	case 376:
		return "ISR"
	case 380:
		return "ITA"
	case 388:
		return "JAM"
	case 392:
		return "JPN"
	case 400:
		return "JOR"
	case 398:
		return "KAZ"
	case 404:
		return "KEN"
	case 296:
		return "KIR"
	case 410:
		return "KOR"
	case 408:
		return "PRK"
	case 414:
		return "KWT"
	case 417:
		return "KGZ"
	case 418:
		return "LAO"
	case 428:
		return "LVA"
	case 422:
		return "LBN"
	case 426:
		return "LSO"
	case 430:
		return "LBR"
	case 434:
		return "LBY"
	case 438:
		return "LIE"
	case 440:
		return "LTU"
	case 442:
		return "LUX"
	case 446:
		return "MAC"
	case 807:
		return "MKD"
	case 450:
		return "MDG"
	case 454:
		return "MWI"
	case 458:
		return "MYS"
	case 462:
		return "MDV"
	case 466:
		return "MLI"
	case 470:
		return "MLT"
	case 584:
		return "MHL"
	case 474:
		return "MTQ"
	case 478:
		return "MRT"
	case 480:
		return "MUS"
	case 175:
		return "MYT"
	case 484:
		return "MEX"
	case 583:
		return "FSM"
	case 498:
		return "MDA"
	case 492:
		return "MCO"
	case 496:
		return "MNG"
	case 500:
		return "MSR"
	case 504:
		return "MAR"
	case 508:
		return "MOZ"
	case 104:
		return "MMR"
	case 516:
		return "NAM"
	case 520:
		return "NRU"
	case 524:
		return "NPL"
	case 528:
		return "NLD"
	case 530:
		return "ANT"
	case 540:
		return "NCL"
	case 554:
		return "NZL"
	case 558:
		return "NIC"
	case 562:
		return "NER"
	case 566:
		return "NGA"
	case 570:
		return "NIU"
	case 574:
		return "NFK"
	case 580:
		return "MNP"
	case 578:
		return "NOR"
	case 512:
		return "OMN"
	case 586:
		return "PAK"
	case 585:
		return "PLW"
	case 275:
		return "PSE"
	case 591:
		return "PAN"
	case 598:
		return "PNG"
	case 600:
		return "PRY"
	case 604:
		return "PER"
	case 608:
		return "PHL"
	case 612:
		return "PCN"
	case 616:
		return "POL"
	case 620:
		return "PRT"
	case 630:
		return "PRI"
	case 634:
		return "QAT"
	case 638:
		return "REU"
	case 642:
		return "ROU"
	case 643:
		return "RUS"
	case 646:
		return "RWA"
	case 654:
		return "SHN"
	case 659:
		return "KNA"
	case 662:
		return "LCA"
	case 666:
		return "SPM"
	case 670:
		return "VCT"
	case 882:
		return "WSM"
	case 674:
		return "SMR"
	case 678:
		return "STP"
	case 682:
		return "SAU"
	case 686:
		return "SEN"
	case 690:
		return "SYC"
	case 694:
		return "SLE"
	case 702:
		return "SGP"
	case 703:
		return "SVK"
	case 705:
		return "SVN"
	case 90:
		return "SLB"
	case 706:
		return "SOM"
	case 710:
		return "ZAF"
	case 239:
		return "SGS"
	case 724:
		return "ESP"
	case 144:
		return "LKA"
	case 729:
		return "SDN"
	case 740:
		return "SUR"
	case 744:
		return "SJM"
	case 748:
		return "SWZ"
	case 752:
		return "SWE"
	case 756:
		return "CHE"
	case 760:
		return "SYR"
	case 158:
		return "TWN"
	case 762:
		return "TJK"
	case 834:
		return "TZA"
	case 764:
		return "THA"
	case 626:
		return "TLS"
	case 768:
		return "TGO"
	case 772:
		return "TKL"
	case 776:
		return "TON"
	case 780:
		return "TTO"
	case 788:
		return "TUN"
	case 792:
		return "TUR"
	case 795:
		return "TKM"
	case 796:
		return "TCA"
	case 798:
		return "TUV"
	case 800:
		return "UGA"
	case 804:
		return "UKR"
	case 784:
		return "ARE"
	case 826:
		return "GBR"
	case 840:
		return "USA"
	case 581:
		return "UMI"
	case 858:
		return "URY"
	case 860:
		return "UZB"
	case 548:
		return "VUT"
	case 336:
		return "VAT"
	case 862:
		return "VEN"
	case 704:
		return "VNM"
	case 92:
		return "VGB"
	case 850:
		return "VIR"
	case 876:
		return "WLF"
	case 732:
		return "ESH"
	case 887:
		return "YEM"
	case 891:
		return "YUG"
	case 894:
		return "ZMB"
	case 716:
		return "ZWE"
	case 4:
		return "AFG"
	case 688:
		return "SRB"
	case 248:
		return "ALA"
	case 535:
		return "BES"
	case 831:
		return "GGY"
	case 832:
		return "JEY"
	case 531:
		return "CUW"
	case 833:
		return "IMN"
	case 652:
		return "BLM"
	case 663:
		return "MAF"
	case 534:
		return "SXM"
	case 499:
		return "MNE"
	case 728:
		return "SSD"
	case 900:
		return "XKX"
	case 998:
		return "None"
	case 999:
		return "International"
	case 999800:
		return "International Freephone"
	case 999870:
		return "Inmarsat"
	case 999875:
		return "Maritime Mobile service"
	case 999878:
		return "Universal Personal Telecommunications services"
	case 999879:
		return "National non-commercial purposes"
	case 999881:
		return "Global Mobile Satellite System"
	case 999882:
		return "International Networks"
	case 999888:
		return "Disaster Relief"
	case 999979:
		return "International Premium Rate Service"
	case 999991:
		return "International Telecommunications Public Correspondence Service"
	}
	return UnknownMsg
}
