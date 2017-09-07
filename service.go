//
// Copyright (c) 2017 Joey <majunjiev@gmail.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package ovirtsdk4

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Service is the interface of all type services.
type Service interface {
	Connection() *Connection
	Path() string
}

// baseService represents the base for all the services of the SDK. It contains the
// utility methods used by all of them.
type baseService struct {
	connection *Connection
	path       string
}

func (service *baseService) Connection() *Connection {
	return service.connection
}

func (service *baseService) Path() string {
	return service.path
}

// CheckFault procoesses error parsing and returns it back
func CheckFault(response *http.Response) error {
	resBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response, reason: %s", err.Error())
	}

	reader := NewXMLReader(resBytes)
	fault, err := XMLFaultReadOne(reader, nil, "")
	if err != nil {
		// If the XML is not a <fault>, just return nil
		if _, ok := err.(XMLTagNotMatchError); ok {
			return nil
		}
		return err
	}
	if fault != nil || response.StatusCode >= 400 {
		return BuildError(response, fault)
	}
	return nil
}

// CheckAction checks if response contains an Action instance
func CheckAction(response *http.Response) (*Action, error) {
	resBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read response, reason: %s", err.Error())
	}

	faultreader := NewXMLReader(resBytes)
	fault, err := XMLFaultReadOne(faultreader, nil, "")
	if err != nil {
		// If the tag mismatches, return the err
		if _, ok := err.(XMLTagNotMatchError); !ok {
			return nil, err
		}
	}
	if fault != nil {
		return nil, BuildError(response, fault)
	}

	actionreader := NewXMLReader(resBytes)
	action, err := XMLActionReadOne(actionreader, nil, "")
	if err != nil {
		// If the tag mismatches, return the err
		if _, ok := err.(XMLTagNotMatchError); !ok {
			return nil, err
		}
	}
	if action != nil {
		if afault, ok := action.Fault(); ok {
			return nil, BuildError(response, afault)
		}
		return action, nil
	}
	return nil, nil
}

// BuildError constructs error
func BuildError(response *http.Response, fault *Fault) error {
	var buffer bytes.Buffer
	if fault != nil {
		if reason, ok := fault.Reason(); ok {
			if buffer.Len() > 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(fmt.Sprintf("Fault reason is \"%s\".", reason))
		}
		if detail, ok := fault.Detail(); ok {
			if buffer.Len() > 0 {
				buffer.WriteString(" ")
			}
			buffer.WriteString(fmt.Sprintf("Fault detail is \"%s\".", detail))
		}
	}
	if response != nil {
		if buffer.Len() > 0 {
			buffer.WriteString(" ")
		}
		buffer.WriteString(fmt.Sprintf("HTTP response code is \"%d\".", response.StatusCode))
		buffer.WriteString(" ")
		buffer.WriteString(fmt.Sprintf("HTTP response message is \"%s\".", response.Status))
	}

	return errors.New(buffer.String())
}
