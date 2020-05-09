package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type WitnessList struct {
	Witness []WitnessMeta `xml:"witness"`
}

type WitnessMeta struct {
	ID      string   `xml:"sameAs,attr"`
	Abbrevs []Abbrev `xml:"abbr"`
}

type Abbrev struct {
	Name       string      `xml:",chardata"`
	Extensions []Extension `xml:"hi"`
}

type Extension struct {
	Name string `xml:",chardata"`
}

type Milestone struct {
	ID   string `xml:"n,attr"`
	Type string `xml:"unit,attr"`
}

type Anchor struct {
	ID string `xml:"id,attr"`
}

type AppData2 struct {
	Type  string `xml:"type,attr"`
	Inner string `xml:",innerxml"`
}

type AppData struct {
	Type       string      `xml:"type,attr"`
	ToAnchor   string      `xml:"to,attr"`
	Variant    []Variants  `xml:"rdg"`
	WitDetails []WitDetail `xml:"witDetail"`
}

type Variants struct {
	VariantWitnesses string `xml:"wit,attr"`
	VariantID        string `xml:"id,attr"`
	VariantText      string `xml:",chardata"`
}

type WitDetail struct {
	Target string `xml:"target,attr"`
	Wit    string `xml:"wit,attr"`
	Detail string `xml:",chardata"`
}

type Witness struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type Alignment struct {
	ID    string
	Token []CTSPassage
}

type CTSPassage struct {
	ID      string
	Passage string
}

var allowedDetail = []string{"ac", "pc"}

var splits = []rune{' ', '‌', '|', '〉'}

func testSplit(char rune) bool {
	for _, v := range splits {
		if v == char {
			return (true)
		}
	}
	return (false)
}

func anyLetters(s string) bool {
	for _, v := range s {
		if unicode.IsLetter(v) {
			return true
		}
	}
	return false
}

func customSplit(passage string) (tokens []string) {
	found := false
	leadingWS := true
	tmpstring := ""

	for _, char := range passage {
		if !unicode.IsLetter(char) && leadingWS {
			tmpstring = tmpstring + string(char)
			continue
		}
		leadingWS = false
		if found == true {
			if testSplit(char) {
				tmpstring = tmpstring + string(char)
				continue
			} else {
				tokens = append(tokens, tmpstring)
				tmpstring = string(char)
				found = false
				leadingWS = true
				continue
			}
		}
		if testSplit(char) {
			found = true
		}
		tmpstring = tmpstring + string(char)
	}
	if !anyLetters(tmpstring) && len(tokens) > 0 {
		tokens[len(tokens)-1] = tokens[len(tokens)-1] + tmpstring
	} else {
		tokens = append(tokens, tmpstring)
	}
	return (tokens)
}

const passageBase = "urn:cts:sktlit:skt0001.nyaya002."

var editionsMap = make(map[string][]CTSPassage)
var witnessMap = make(map[string]bool)
var secWitnessMap = make(map[string]bool)
var siglaMap = make(map[string]string)
var alignments = []Alignment{}

var witnessRange = make(map[string]map[string]bool)
var witBool = make(map[string]bool)

func establishWit() {
	log.Println("Establish Witness Range...")
	currentMilestone := ""
	actualText := false
	appIsOpen := false
	witsearch := regexp.MustCompile(`#M\d+[^\s,"]`)
	bytexml, err := os.Open("2020_02_19_Collation_NBh 3.xml")
	if err != nil {
		panic(err)
	}
	defer bytexml.Close()
	currentChapter := "prelim"
	decoder := xml.NewDecoder(bytexml)
	count := 0
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "milestone":
				var milestone Milestone
				decoder.DecodeElement(&milestone, &se)
				if milestone.Type == "chapter" && milestone.Type != currentMilestone {
					currentChapter = milestone.ID
					actualText = true
				}
			case "anchor":
				if !actualText {
					// break
				}
				witnessRange[currentChapter] = make(map[string]bool)
				for k, v := range witBool {
					witnessRange[currentChapter][k] = v
				}
				appIsOpen = false
			case "app":
				if !appIsOpen {
					if currentChapter == "prelim" {
						currentChapter = "3.1.1"
					}
				}
				appIsOpen = true
				var appdata AppData2
				decoder.DecodeElement(&appdata, &se)
				if appdata.Type != "a1" {
					break
				}
				switch {
				case strings.Contains(appdata.Inner, "witStart"):
					log.Println("Witness start at", currentChapter)
					indparts := strings.Split(appdata.Inner, "witStart")
					for i, v := range indparts {
						if i == len(indparts)-1 {
							break
						}
						witsl := strings.Split(v, "rdg")
						for _, v2 := range witsl {
							witstrs := witsearch.FindAllString(v2, -1)
							for _, v3 := range witstrs {
								newwit := strings.Replace(v3, "#", "", -1)
								witBool[newwit] = true
							}
						}
					}
				case strings.Contains(appdata.Inner, "witEnd"):
					log.Println("Witness end at", currentChapter)
					indparts := strings.Split(appdata.Inner, "witEnd")
					for i, v := range indparts {
						if i == len(indparts)-1 {
							break
						}
						witsl := strings.Split(v, "rdg")
						for _, v2 := range witsl {
							witstrs := witsearch.FindAllString(v2, -1)
							for _, v3 := range witstrs {
								newwit := strings.Replace(v3, "#", "", -1)
								witBool[newwit] = false
							}
						}
					}
				default:
					break
				}
			}
		}
	}
	log.Println("Done.", count)
}

func main() {
	establishWit()
	bytexml, err := os.Open("2020_02_19_Collation_NBh 3.xml")
	// 2020_02_06_Collation_NBh3.xml
	if err != nil {
		panic(err)
	}
	defer bytexml.Close()
	report, err := os.Create("report.txt")
	if err != nil {
		panic(err)
	}
	defer report.Close()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	var positionMap = map[string]map[string]string{}
	var secPositionMap = map[string]map[string]string{}
	lemmaCount := 1
	basetext := []string{}
	passageURNs := []string{}
	passageURN := "start"
	currentMilestone := ""
	actualText := false
	appIsOpen := false
	appURN := ""

	currentChapter := "prelim"
	decoder := xml.NewDecoder(bytexml)
	bodyOpen := false
	noteOpen := false
	passageBuffer := ""
	spaceReg := regexp.MustCompile(`\s+`)

	// flushed := false

	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.EndElement:
			switch se.Name.Local {
			case "note":
				noteOpen = false
			case "body":
				bodyOpen = false
				break
			}
		case xml.StartElement:
			switch se.Name.Local {
			case "listWit":
				if bodyOpen == false {
					witlist := WitnessList{}
					decoder.DecodeElement(&witlist, &se)
					for _, v := range witlist.Witness {
						key := strings.TrimSpace(v.ID)
						value := []string{}
						for _, v2 := range v.Abbrevs {
							firstid := v2.Name
							resolution := []string{}
							firstid = strings.ReplaceAll(firstid, "^!", "Note")
							firstid = strings.ReplaceAll(firstid, "(", "")
							firstid = strings.ReplaceAll(firstid, ")", "")
							firstid = strings.TrimSpace(firstid)
							firstid = spaceReg.ReplaceAllString(firstid, "_")
							// You almost cannot see the difference, but there is one!!
							// firstid = strings.ReplaceAll(firstid, " ", "")
							firstid = strings.ReplaceAll(firstid, " ", "")
							if firstid != "" {
								value = append(value, firstid)
							}
							resolution = append(resolution, firstid)
							for _, v3 := range v2.Extensions {
								secondid := v3.Name
								secondid = strings.ReplaceAll(secondid, "^!", "Note")
								secondid = strings.ReplaceAll(secondid, "(", "")
								secondid = strings.ReplaceAll(secondid, ")", "")
								secondid = strings.TrimSpace(secondid)
								secondid = spaceReg.ReplaceAllString(secondid, "_")
								secondid = strings.ReplaceAll(secondid, " _", "_")
								if secondid != "" {
									value = append(value, secondid)
								}
							}
						}
						if key != "" {
							siglaMap[key] = strings.Join(value, "_")
						}
					}
				}
			case "milestone":
				var milestone Milestone
				decoder.DecodeElement(&milestone, &se)
				if milestone.Type == "chapter" && milestone.Type != currentMilestone {
					currentChapter = milestone.ID
					lemmaCount = 1
					actualText = true
				}
			case "anchor":
				if !actualText {
					break
				}
				var anchor Anchor
				decoder.DecodeElement(&anchor, &se)

				passageURN = currentChapter + "." + fmt.Sprintf("%d", lemmaCount)
				basetext = append(basetext, passageBuffer)
				passageURNs = append(passageURNs, passageURN)
				passageBuffer = ""
				// flushed = true
				lemmaCount++
				appIsOpen = false
			case "app":
				if !appIsOpen {
					if currentChapter == "prelim" {
						currentChapter = "3.1.1"
					}
					appURN = currentChapter + "." + fmt.Sprintf("%d", lemmaCount)
				}
				appIsOpen = true
				var appdata AppData
				decoder.DecodeElement(&appdata, &se)
				for _, variant := range appdata.Variant {
					witNames := strings.Split(variant.VariantWitnesses, " ")
					switch {
					case variant.VariantID != "":
						for _, witDetails := range appdata.WitDetails {
							if witDetails.Target != variant.VariantID {
								continue
							}
							witDetStr := witDetails.Wit
							witDetStr = strings.Replace(witDetStr, " ", "", -1)
							witDetStr = strings.Replace(witDetStr, "#", "", -1)
							witDetStr = strings.Replace(witDetStr, "\n", "", -1)
							detail := strings.TrimSpace(witDetails.Detail)
							switch detail {
							case "pc":
								newkey := strings.Join([]string{witDetStr, detail}, "_")
								newvalue := strings.Join([]string{siglaMap[witDetStr], detail}, "_")
								siglaMap[newkey] = newvalue
								readingvalue := variant.VariantText
								readingvalue = strings.Replace(readingvalue, "\n", "", -1)
								if strings.TrimSpace(readingvalue) == "" {
									readingvalue = "[[om.]]"
								}
								if len(secPositionMap[appURN]) == 0 {
									secPositionMap[appURN] = make(map[string]string)
								}
								secWitnessMap[newvalue] = true
								secPositionMap[appURN][newvalue] = readingvalue
							case "vl":
								switch appdata.Type {
								case "a6":
									newkey := strings.Join([]string{witDetStr, detail}, "_")
									newvalue := strings.Join([]string{siglaMap[witDetStr], detail}, "_")
									siglaMap[newkey] = newvalue
									readingvalue := variant.VariantText
									readingvalue = strings.Replace(readingvalue, "\n", "", -1)
									if strings.TrimSpace(readingvalue) == "" {
										readingvalue = "[[om.]]"
									}
									if len(positionMap[appURN]) == 0 {
										positionMap[appURN] = make(map[string]string)
									}
									witnessMap[newvalue] = true
									positionMap[appURN][newvalue] = readingvalue
								default:
									newkey := strings.Join([]string{witDetStr, detail}, "_")
									newvalue := strings.Join([]string{siglaMap[witDetStr], detail}, "_")
									siglaMap[newkey] = newvalue
									readingvalue := variant.VariantText
									readingvalue = strings.Replace(readingvalue, "\n", "", -1)
									if strings.TrimSpace(readingvalue) == "" {
										readingvalue = "[[om.]]"
									}
									if len(secPositionMap[appURN]) == 0 {
										secPositionMap[appURN] = make(map[string]string)
									}
									secWitnessMap[newvalue] = true
									secPositionMap[appURN][newvalue] = readingvalue
								}
							default:
								if strings.Contains(detail, "pc") {
									newkey := strings.Join([]string{witDetStr, "2pc"}, "_")
									newvalue := strings.Join([]string{siglaMap[witDetStr], "2pc"}, "_")
									siglaMap[newkey] = newvalue
									readingvalue := variant.VariantText
									readingvalue = strings.Replace(readingvalue, "\n", "", -1)
									if strings.TrimSpace(readingvalue) == "" {
										readingvalue = "[[om.]]"
									}
									if len(positionMap[appURN]) == 0 {
										positionMap[appURN] = make(map[string]string)
									}
									witnessMap[newvalue] = true
									positionMap[appURN][newvalue] = readingvalue
								} else {
									// still save without addon
									readingvalue := variant.VariantText
									readingvalue = strings.Replace(readingvalue, "\n", "", -1)
									if strings.TrimSpace(readingvalue) == "" {
										readingvalue = "[[om.]]"
									}
									if len(positionMap[appURN]) == 0 {
										positionMap[appURN] = make(map[string]string)
									}
									resolSigl := siglaMap[witDetStr]
									witnessMap[resolSigl] = true
									positionMap[appURN][resolSigl] = readingvalue
								}
							}
						}
					default:
						for _, witname := range witNames {
							witDetStr := witname
							witDetStr = strings.Replace(witDetStr, " ", "", -1)
							witDetStr = strings.Replace(witDetStr, "#", "", -1)
							witDetStr = strings.Replace(witDetStr, "\n", "", -1)
							readingvalue := variant.VariantText
							readingvalue = strings.Replace(readingvalue, "\n", "", -1)
							if strings.TrimSpace(readingvalue) == "" {
								readingvalue = "[[om.]]"
							}
							if len(positionMap[appURN]) == 0 {
								positionMap[appURN] = make(map[string]string)
							}
							resolSigl := siglaMap[witDetStr]
							witnessMap[resolSigl] = true
							positionMap[appURN][resolSigl] = readingvalue
						}

					}
				}
			case "note":
				noteOpen = true
			case "body":
				bodyOpen = true
			}
		case xml.CharData:
			if !actualText {
				break
			}
			if bodyOpen && !noteOpen {
				// flushed = false
				outputString := string(se)
				outputString = strings.Replace(outputString, "\n", " ", -1)
				passageBuffer = passageBuffer + outputString
				// outputString = strings.TrimSpace(outputString)
				// if outputString != "" {
				// }
			}
		}
	}

	basetext = append(basetext, passageBuffer)
	numID := lemmaCount + 1
	passageURN = currentChapter + "." + fmt.Sprintf("%d", numID)
	passageURNs = append(passageURNs, passageURN)
	lemmaCount++

	log.Println(len(basetext))
	log.Println(len(passageURNs))

	for key, value := range basetext {
		keyStr := passageURNs[key]
		report.WriteString("---------------------------------------------")
		report.WriteString("\n")
		alignmentID := "urn:cite2:ducat:alignments.temp:" + keyStr
		editionURN := passageBase + "DFG.token:"
		tmpalignment := Alignment{ID: alignmentID}
		for index, element := range customSplit(value) {
			idPassage := editionURN + keyStr + "_" + strconv.Itoa(index+1)
			passages := editionsMap[editionURN]
			tmpPassage := CTSPassage{ID: idPassage, Passage: element}
			passages = append(passages, tmpPassage)
			tmpalignment.Token = append(tmpalignment.Token, tmpPassage)
			editionsMap[editionURN] = passages
		}
		report.WriteString(fmt.Sprintln("Position: ", keyStr, "Reading:", value))
		report.WriteString("\n")

		report.WriteString("Variants:")
		report.WriteString("\n")
		for witkey := range witnessMap {
			witnessURN := passageBase + witkey + ".token:"
			reading, ok := positionMap[keyStr][witkey]
			if !ok {
				reading = "[[NA]]"
			}
			for index, element := range customSplit(reading) {
				idPassage := witnessURN + keyStr + "_" + strconv.Itoa(index+1)
				passages := editionsMap[witnessURN]
				// only for testing
				if element == "" {
					element = " "
				}
				//
				tmpPassage := CTSPassage{ID: idPassage, Passage: element}
				passages = append(passages, tmpPassage)
				tmpalignment.Token = append(tmpalignment.Token, tmpPassage)
				editionsMap[witnessURN] = passages
			}
			report.WriteString(fmt.Sprintln(witkey, "Reading:", reading))
		}
		alignments = append(alignments, tmpalignment)
	}
	for _, v := range editionsMap {
		for i := range v {
			report.WriteString(fmt.Sprintln(v[i]))
			break
		}
		break
	}

	for k, v := range siglaMap {
		report.WriteString(fmt.Sprintln("key:", k, "value:", v))
	}
	report.WriteString(fmt.Sprintln("Parsed", len(alignments), "lemmata..."))
	log.Println("Parsed", len(alignments), "lemmata...")
	log.Println(len(basetext))

	for k, v := range witnessRange {
		report.WriteString("______________________________")
		report.WriteString(fmt.Sprintln("Passage:", k))
		for k2, v2 := range v {
			report.WriteString(fmt.Sprintln("key:", k2, "value:", v2))
		}
	}
	writeCEX()
}

func writeCEX() {
	f, err := os.Create("output.cex")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// cexversion
	f.WriteString("#!cexversion\n")
	f.WriteString("3.0\n\n")

	f.WriteString("#!citelibrary\n")
	f.WriteString("name#CITE Library generated by Brucheion\n")
	f.WriteString("urn#urn:cite2:cex:brucheion.version1:123\n")
	f.WriteString("license#CC Share Alike.\n\n")

	// ctscatalog
	f.WriteString("#!ctscatalog\n")
	f.WriteString("urn#citationScheme#groupName#workTitle#versionLabel#exemplarLabel#online#language")
	f.WriteString("\n")

	editionURN := passageBase + "DFG.token:"
	f.WriteString(editionURN)
	f.WriteString("#")
	f.WriteString("NyayaScheme")
	f.WriteString("#")
	f.WriteString("GroupName")
	f.WriteString("#")
	f.WriteString("WorkTitle")
	f.WriteString("#")
	f.WriteString("VersionLabel")
	f.WriteString("#")
	f.WriteString("Brucheion-Tokenised")
	f.WriteString("#")
	f.WriteString("TRUE")
	f.WriteString("#")
	f.WriteString("san")
	f.WriteString("\n")

	for witkey := range witnessMap {
		witnessURN := passageBase + witkey + ".token:"
		f.WriteString(witnessURN)
		f.WriteString("#")
		f.WriteString("NyayaScheme")
		f.WriteString("#")
		f.WriteString("GroupName")
		f.WriteString("#")
		f.WriteString("WorkTitle")
		f.WriteString("#")
		f.WriteString("VersionLabel")
		f.WriteString("#")
		f.WriteString("Brucheion-Tokenised")
		f.WriteString("#")
		f.WriteString("TRUE")
		f.WriteString("#")
		f.WriteString("san")
		f.WriteString("\n")
	}
	f.WriteString("\n")
	f.WriteString("#!ctsdata\n")

	for _, edition := range editionsMap {
		for passageIndex := range edition {
			f.WriteString(edition[passageIndex].ID)
			f.WriteString("#")
			f.WriteString(edition[passageIndex].Passage)
			f.WriteString("\n")
		}
	}
	f.WriteString("\n")

	f.WriteString("#!datamodels\n")
	f.WriteString("Collection#Model#Label#Description\n")
	f.WriteString("urn:cite2:ducat:alignments.temp:#urn:cite2:cite:datamodels.v1:alignment#Text Alignment Model#The CITE model for text alignment. See documentation at <https://eumaeus.github.io/citealign/>.\n")
	f.WriteString("\n")

	f.WriteString("#!citecollections\n")
	f.WriteString("URN#Description#Labelling property#Ordering property#License\n")
	f.WriteString("urn:cite2:ducat:alignments.temp:#Citation Alignments#urn:cite2:ducat:alignments.temp.label:##CC-BY 3.0\n")
	f.WriteString("\n")

	f.WriteString("#!citeproperties\n")
	f.WriteString("Property#Label#Type#Authority list\n")
	f.WriteString("urn:cite2:ducat:alignments.temp.urn:#Alignment Record#Cite2Urn#\n")
	f.WriteString("urn:cite2:ducat:alignments.temp.label:#Label#String#\n")
	f.WriteString("urn:cite2:ducat:alignments.temp.description:#Description#String#\n")
	f.WriteString("urn:cite2:ducat:alignments.temp.editor:#Editor#String#\n")
	f.WriteString("urn:cite2:ducat:alignments.temp.date:#Date#String#\n")
	f.WriteString("\n")

	f.WriteString("#!citedata\n")
	f.WriteString("urn#label#description#editor#date\n")
	count := 1
	for _, alignment := range alignments {
		// testing
		// if alignment.ID == "urn:cite2:ducat:alignments.temp:3.1.1.50" {
		// 	break
		// }
		//
		f.WriteString(alignment.ID)
		f.WriteString("#")
		alignLabel := "Alignment " + strconv.Itoa(count)
		f.WriteString(alignLabel)
		f.WriteString("#")
		f.WriteString("Textual Alignment")
		f.WriteString("#")
		f.WriteString("Brucheion User")
		f.WriteString("#")
		f.WriteString("Sun, 19 Apr 2020 12:30:32 GMT")
		f.WriteString("\n")
		count++
	}
	f.WriteString("\n")

	f.WriteString("#!relations\n")
	for _, alignment := range alignments {
		// testing
		// if alignment.ID == "urn:cite2:ducat:alignments.temp:3.1.1.50" {
		// 	break
		// }
		//
		for _, passage := range alignment.Token {
			f.WriteString(alignment.ID)
			f.WriteString("#urn:cite2:cite:verbs.v1:aligns#")
			f.WriteString(passage.ID)
			f.WriteString("\n")
		}
	}
	f.WriteString("\n")

}
