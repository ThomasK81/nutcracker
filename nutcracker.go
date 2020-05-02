package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
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

type AppData struct {
	Variant    []Variants  `xml:"rdg"`
	WitDetails []WitDetail `xml:"witDetail"`
}

type Variants struct {
	VariantWitnesses string `xml:"wit,attr"`
	VariantID        string `xml:"id,attr"`
	VariantText      string `xml:",chardata"`
}

type WitDetail struct {
	Target string   `xml:"target,attr"`
	Wit    string   `xml:"wit,attr"`
	Detail []string `xml:"hi"`
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

var allowedDetail = []string{"ac", "pc", "v1"}

var splits = []rune{' ', '‌'}
var splitTests = []rune{' ', '‌', '|'}

func testSplit(char rune) bool {
	for _, v := range splits {
		if v == char {
			return (true)
		}
	}
	return (false)
}

func testSplit2(char rune) bool {
	for _, v := range splitTests {
		if v == char {
			return (true)
		}
	}
	return (false)
}

func customSplit(passage string) (tokens []string) {
	found := false
	tmpstring := ""
	for _, char := range passage {
		if !testSplit2(char) && found == true {
			tokens = append(tokens, tmpstring)
			tmpstring = ""
			found = false
		}
		tmpstring = tmpstring + string(char)
		if testSplit(char) {
			found = true
		}
	}
	tokens = append(tokens, tmpstring)
	return (tokens)
}

const passageBase = "urn:cts:sktlit:skt0001.nyaya002."

var editionsMap = make(map[string][]CTSPassage)
var witnessMap = make(map[string]bool)
var siglaMap = make(map[string][]string)
var alignments = []Alignment{}

func main() {
	bytexml, err := os.Open("2020_02_06_Collation_NBh3.xml")
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
	lemmaCount := 0
	basetext := []string{}
	passageURNs := []string{}
	passageURN := "start"
	currentChapter := "PreLim"
	decoder := xml.NewDecoder(bytexml)
	bodyOpen := false
	noteOpen := false
	passageBuffer := ""
	spaceReg := regexp.MustCompile(`\s+`)
	actualText := false
	mileCount := 0
	for {
		// Read tokens from the XML document in a stream.
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
							firstid = strings.ReplaceAll(firstid, "^!", "_")
							firstid = strings.ReplaceAll(firstid, "(", "")
							firstid = strings.ReplaceAll(firstid, ")", "")
							firstid = strings.TrimSpace(firstid)
							firstid = spaceReg.ReplaceAllString(firstid, "_")
							// You almost cannot see the difference, but there is one!!
							// firstid = strings.ReplaceAll(firstid, " ", "")
							firstid = strings.ReplaceAll(firstid, " ", "")
							log.Println(firstid)
							if firstid != "" {
								value = append(value, firstid)
							}
							for _, v3 := range v2.Extensions {
								secondid := v3.Name
								secondid = strings.ReplaceAll(secondid, "^!", "_")
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
							siglaMap[key] = value
						}
					}
				}
			case "milestone":
				var milestone Milestone
				decoder.DecodeElement(&milestone, &se)
				if milestone.Type == "chapter" {
					currentChapter = milestone.ID
					lemmaCount = 0
					if mileCount > 0 {
						actualText = true
					}
					mileCount++
				}
			case "app":
				if !actualText {
					break
				}
				basetext = append(basetext, passageBuffer)
				numID := lemmaCount + 1
				passageURN = currentChapter + "." + fmt.Sprintf("%d", numID)
				passageURNs = append(passageURNs, passageURN)
				lemmaCount++
				passageBuffer = ""

				var appdata AppData
				decoder.DecodeElement(&appdata, &se)
				for i := range appdata.Variant {
					old := ""
					modified := ""
					witNames := strings.Split(appdata.Variant[i].VariantWitnesses, " ")
					if appdata.Variant[i].VariantID != "" {
						for j := range appdata.WitDetails {
							if appdata.WitDetails[j].Target != appdata.Variant[i].VariantID {
								continue
							}
							old = appdata.WitDetails[j].Wit
							old = strings.Replace(old, " ", "", -1)
							old = strings.Replace(old, "#", "", -1)
							old = strings.Replace(old, "\n", "", -1)
							cleanedDetails := []string{}
							for _, hiElem := range appdata.WitDetails[j].Detail {
								for _, iDetail := range allowedDetail {
									if iDetail == hiElem {
										cleanedDetails = append(cleanedDetails, hiElem)
									}
								}
							}
							if len(cleanedDetails) > 0 {
								added := strings.Join(cleanedDetails, "-")
								modified = old + "__" + added
							} else {
								modified = old
							}

						}
					}
					for _, witname := range witNames {
						witname = strings.Replace(witname, " ", "", -1)
						witname = strings.Replace(witname, "#", "", -1)
						witname = strings.Replace(witname, "\n", "", -1)
						if witname != "" {
							if witname == old {
								witname = modified
							}
							key2 := witname
							// testing
							// if len(witnessMap) > 3 {
							// 	continue
							// }
							//

							value := appdata.Variant[i].VariantText
							value = strings.Replace(value, "\n", "", -1)
							if strings.TrimSpace(value) == "" {
								value = "[[om.]]"
							}
							if len(positionMap[passageURN]) == 0 {
								positionMap[passageURN] = make(map[string]string)
							}
							keylist := []string{}
							if strings.Contains(key2, "__") {
								resolved := strings.Split(key2, "__")
								key3 := resolved[0]
								ext := resolved[1]
								oddkeys := siglaMap[key3]
								for _, keyv := range oddkeys {
									newkey := strings.Join([]string{keyv, ext}, "_")
									keylist = append(keylist, newkey)
								}
							} else {
								keylist = siglaMap[key2]
							}
							for _, keyv := range keylist {
								witnessMap[keyv] = true
								positionMap[passageURN][keyv] = value
							}
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
				outputString := string(se)
				outputString = strings.Replace(outputString, "\n", " ", -1)
				passageBuffer = passageBuffer + outputString
				// outputString = strings.TrimSpace(outputString)
				// if outputString != "" {
				// }
			}
		}
	}

	for key, value := range basetext {
		keyStr := passageURNs[key]
		// if strings.HasPrefix(keyStr, "PreLim") {
		// 	continue
		// }
		report.WriteString("---------------------------------------------")
		report.WriteString("\n")
		alignmentID := "urn:cite2:ducat:alignments.temp:" + keyStr
		editionURN := passageBase + "DFG.token:"
		tmpalignment := Alignment{ID: alignmentID}
		for index, element := range customSplit(value) {
			idPassage := editionURN + keyStr + "_" + strconv.Itoa(index+1)
			passages := editionsMap[editionURN]
			// only for testing
			if element == "" {
				element = " "
			}
			//
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
				reading = value
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
		report.WriteString(fmt.Sprintln("key:", k))
		for i, v2 := range v {
			report.WriteString(fmt.Sprintln("value:", i, ":", v2))
		}
	}
	report.WriteString(fmt.Sprintln("Parsed", len(alignments), "lemmata..."))
	log.Println("Parsed", len(alignments), "lemmata...")
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
